package awsprov

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/molecule-man/claws/cloudprov"
)

// AwsProvider is wrapper over aws sdk
type AwsProvider struct {
	cf *cloudformation.CloudFormation
}

// New creates a new AwsProvider
func New() AwsProvider {
	sess := session.Must(session.NewSession())
	cf := cloudformation.New(sess)

	return AwsProvider{
		cf: cf,
	}
}

// ValidateTemplate validates the stack template
// Returns parameters accepted by stack
func (ap *AwsProvider) ValidateTemplate(tplBody string) ([]string, error) {
	v := &cloudformation.ValidateTemplateInput{
		TemplateBody: aws.String(tplBody),
	}
	out, err := ap.cf.ValidateTemplate(v)

	if err != nil {
		return []string{}, err
	}

	params := make([]string, 0, len(out.Parameters))

	for _, p := range out.Parameters {
		params = append(params, *p.ParameterKey)
	}

	return params, nil
}

// StackExists returns true if the stack exists
func (ap *AwsProvider) StackExists(stackName string) (bool, error) {
	info, err := ap.cf.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})

	if err == nil {
		for _, stack := range info.Stacks {
			if *stack.StackStatus == cloudformation.StackStatusReviewInProgress {
				return false, nil
			}
		}
	} else {
		if strings.Contains(err.Error(), "does not exist") {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// CreateChangeSet creates new change set
// Returns change set ID
func (ap *AwsProvider) CreateChangeSet(stackName string, tplBody string, params map[string]string, op cloudprov.ChangeSetOperation) (string, error) {
	awsParams := make([]*cloudformation.Parameter, 0, len(params))

	for k, v := range params {
		awsParams = append(awsParams, &cloudformation.Parameter{
			ParameterKey:   aws.String(k),
			ParameterValue: aws.String(v),
		})
	}

	awsOperation := cloudformation.ChangeSetTypeCreate

	if op == cloudprov.UpdateOperation {
		awsOperation = cloudformation.ChangeSetTypeUpdate
	}

	createOut, err := ap.cf.CreateChangeSet(&cloudformation.CreateChangeSetInput{
		ChangeSetType: aws.String(awsOperation),
		ChangeSetName: aws.String("chst-" + strconv.FormatInt(time.Now().UnixNano(), 10)),
		TemplateBody:  aws.String(tplBody),
		StackName:     aws.String(stackName),
		Parameters:    awsParams,
	})

	if err != nil {
		return "", err
	}

	return *createOut.Id, nil
}

// WaitChangeSetCreated blocks runtime until the change set is not created
func (ap *AwsProvider) WaitChangeSetCreated(ID string) error {
	return ap.cf.WaitUntilChangeSetCreateCompleteWithContext(aws.BackgroundContext(), &cloudformation.DescribeChangeSetInput{
		ChangeSetName: aws.String(ID),
	}, func(w *request.Waiter) {
		w.Delay = request.ConstantWaiterDelay(time.Second)
	})

}

// ChangeSetChanges returns info about changes to be applied to stack
func (ap *AwsProvider) ChangeSetChanges(ID string) ([]cloudprov.Change, error) {
	setInfo, err := ap.cf.DescribeChangeSet(&cloudformation.DescribeChangeSetInput{
		ChangeSetName: aws.String(ID),
	})

	if err != nil {
		return []cloudprov.Change{}, err
	}

	if *setInfo.Status == cloudformation.ChangeSetStatusFailed {
		if *setInfo.StatusReason == "The submitted information didn't contain changes. Submit different information to create a change set." {
			return []cloudprov.Change{}, cloudprov.ErrNoChange
		}
		return []cloudprov.Change{}, errors.New(*setInfo.StatusReason)
	}

	changes := make([]cloudprov.Change, 0, len(setInfo.Changes))

	for _, c := range setInfo.Changes {
		awsChange := c.ResourceChange
		ch := cloudprov.Change{
			Action:            *awsChange.Action,
			ResourceType:      *awsChange.ResourceType,
			LogicalResourceID: *awsChange.LogicalResourceId,
		}
		if awsChange.Replacement != nil && *awsChange.Replacement == "True" {
			ch.ReplacementNeeded = true
		}

		changes = append(changes, ch)
	}

	return changes, nil
}

// ExecuteChangeSet executes change set identified by ID
func (ap *AwsProvider) ExecuteChangeSet(ID string) error {
	_, err := ap.cf.ExecuteChangeSet(&cloudformation.ExecuteChangeSetInput{
		ChangeSetName: aws.String(ID),
	})
	return err
}

// WaitStack blocks runtime until the stack is not created
func (ap *AwsProvider) WaitStack(stackName string) error {
	stackInput := cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	}

	return ap.cf.WaitUntilStackUpdateCompleteWithContext(aws.BackgroundContext(), &stackInput, func(w *request.Waiter) {
		w.Delay = request.ConstantWaiterDelay(time.Second)
	})
}

// StackEvents returns stack events. The latest events appear first
func (ap *AwsProvider) StackEvents(stackName string) ([]cloudprov.StackEvent, error) {
	awsEvents, err := ap.cf.DescribeStackEvents(&cloudformation.DescribeStackEventsInput{
		StackName: aws.String(stackName),
	})

	if err != nil {
		return []cloudprov.StackEvent{}, err
	}

	events := make([]cloudprov.StackEvent, len(awsEvents.StackEvents))

	for i, e := range awsEvents.StackEvents {
		events[i] = cloudprov.StackEvent{
			ID:                *e.EventId,
			ResourceType:      fromAwsString(e.ResourceType),
			Status:            fromAwsString(e.ResourceStatus),
			LogicalResourceID: fromAwsString(e.LogicalResourceId),
			StatusReason:      fromAwsString(e.ResourceStatusReason),
		}
	}

	return events, nil
}

func fromAwsString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
