package claws

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

//ErrNoChange is error
var ErrNoChange = errors.New("No changes")

// ChangeSet represents aws changeset
type ChangeSet struct {
	Changes   []Change
	StackName string
	ID        *string
	operation string
	cf        *cloudformation.CloudFormation
}

// Change is a change that is applied to aws stack
type Change struct {
	Action            string
	ResourceType      string
	LogicalResourceID string
	ReplacementNeeded bool
}

// New creates a new ChangeSet
func New(stackName string, tplBody string, userParams map[string]string) (*ChangeSet, error) {
	sess := session.Must(session.NewSession())
	cf := cloudformation.New(sess)

	v := &cloudformation.ValidateTemplateInput{
		TemplateBody: aws.String(tplBody),
	}
	out, err := cf.ValidateTemplate(v)

	if err != nil {
		return nil, err
	}

	params := make(map[string]string, len(out.Parameters))

	for _, p := range out.Parameters {
		if v, ok := userParams[*p.ParameterKey]; ok {
			params[*p.ParameterKey] = v
		}
	}

	chSet := &ChangeSet{
		StackName: stackName,
		operation: cloudformation.ChangeSetTypeUpdate,
		cf:        cf,
	}

	err = chSet.initialize(tplBody, params)
	return chSet, err
}

// Exec executes the ChangeSet
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

// EventsTracker creates a new event tracker for the executed stack
func (cs *ChangeSet) EventsTracker() EventsTracker {
	return EventsTracker{
		cf:        cs.cf,
		stackName: cs.StackName,
	}
}

func (cs *ChangeSet) initialize(tplBody string, params map[string]string) error {
	info, err := cs.cf.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(cs.StackName),
	})

	if err == nil {
		for _, stack := range info.Stacks {
			if *stack.StackStatus == cloudformation.StackStatusReviewInProgress {
				cs.operation = cloudformation.ChangeSetTypeCreate
			}
		}
	} else {
		if strings.Contains(err.Error(), "does not exist") {
			cs.operation = cloudformation.ChangeSetTypeCreate
		} else {
			return err
		}
	}

	awsParams := make([]*cloudformation.Parameter, 0, len(params))

	for k, v := range params {
		awsParams = append(awsParams, &cloudformation.Parameter{
			ParameterKey:   aws.String(k),
			ParameterValue: aws.String(v),
		})
	}

	createOut, err := cs.cf.CreateChangeSet(&cloudformation.CreateChangeSetInput{
		ChangeSetType: aws.String(cs.operation),
		ChangeSetName: aws.String("chst-" + strconv.FormatInt(time.Now().UnixNano(), 10)),
		TemplateBody:  aws.String(tplBody),
		StackName:     aws.String(cs.StackName),
		Parameters:    awsParams,
	})

	if err != nil {
		return err
	}

	cs.ID = createOut.Id

	if _, err = cs.describe(); err != nil {
		return err
	}

	err = cs.cf.WaitUntilChangeSetCreateCompleteWithContext(aws.BackgroundContext(), &cloudformation.DescribeChangeSetInput{
		ChangeSetName: createOut.Id,
	}, func(w *request.Waiter) {
		w.Delay = request.ConstantWaiterDelay(time.Second)
	})

	if err != nil {
		return err
	}

	chSetInfo, err := cs.describe()

	if err != nil {
		return err
	}
	cs.Changes = extractChangesFromAwsResponse(chSetInfo.Changes)

	return nil
}

func extractChangesFromAwsResponse(awsChanges []*cloudformation.Change) []Change {
	changes := make([]Change, 0, len(awsChanges))
	for _, c := range awsChanges {
		awsChange := c.ResourceChange
		ch := Change{
			Action:            *awsChange.Action,
			ResourceType:      *awsChange.ResourceType,
			LogicalResourceID: *awsChange.LogicalResourceId,
		}
		if awsChange.Replacement != nil && *awsChange.Replacement == "True" {
			ch.ReplacementNeeded = true
		}

		changes = append(changes, ch)
	}

	return changes
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
