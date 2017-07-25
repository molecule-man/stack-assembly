package claws

import (
	"errors"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

type ChangeSet struct {
	Changes   []Change
	StackName string
	ID        *string
	operation string
	cf        *cloudformation.CloudFormation
}

type Change struct {
	Action            string
	ResourceType      string
	LogicalResourceId string
	ReplacementNeeded bool
}

func (cs *ChangeSet) describe() (*cloudformation.DescribeChangeSetOutput, error) {
	setInfo, err := cs.cf.DescribeChangeSet(&cloudformation.DescribeChangeSetInput{
		ChangeSetName: cs.ID,
	})

	if err != nil {
		return setInfo, err
	}

	if *setInfo.Status == cloudformation.ChangeSetStatusFailed {
		if *setInfo.StatusReason == "The submitted information didn't contain changes. Submit different information to create a change set." {
			return setInfo, ErrNoChange
		}
		return setInfo, errors.New(*setInfo.StatusReason)
	}

	return setInfo, nil
}

func (cs *ChangeSet) EventsTracker() EventsTracker {
	return EventsTracker{
		cf:        cs.cf,
		stackName: cs.StackName,
	}
}

func (cs *ChangeSet) Exec() error {
	_, err := cs.cf.ExecuteChangeSet(&cloudformation.ExecuteChangeSetInput{
		ChangeSetName: cs.ID,
	})

	if err != nil {
		return err
	}

	stackInput := cloudformation.DescribeStacksInput{
		StackName: aws.String(cs.StackName),
	}

	return cs.cf.WaitUntilStackUpdateCompleteWithContext(aws.BackgroundContext(), &stackInput, func(w *request.Waiter) {
		w.Delay = request.ConstantWaiterDelay(time.Second)
	})
}
