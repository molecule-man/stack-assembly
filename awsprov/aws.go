package awsprov

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/molecule-man/claws/claws"
)

const noChangeStatus = "The submitted information didn't contain changes. " +
	"Submit different information to create a change set."

// AwsProvider is wrapper over aws sdk
type AwsProvider struct {
	cf *cloudformation.CloudFormation
}

// New creates a new AwsProvider
func New() *AwsProvider {
	sess := session.Must(session.NewSession())
	cf := cloudformation.New(sess)

	return &AwsProvider{
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

// StackOutputs returns the "outputs" of a stack identified by stackName
func (ap *AwsProvider) StackOutputs(stackName string) (map[string]string, error) {
	outputs := make(map[string]string)

	resp, err := ap.cf.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})

	if err != nil {
		return outputs, err
	}

	for _, s := range resp.Stacks {
		for _, o := range s.Outputs {
			outputs[*o.OutputKey] = *o.OutputValue
		}
	}

	return outputs, nil
}

// CreateChangeSet creates new change set
// Returns change set ID
func (ap *AwsProvider) CreateChangeSet(stackName string, tplBody string, params map[string]string) (string, error) {
	exists, err := ap.StackExists(stackName)

	if err != nil {
		return "", err
	}

	operation := cloudformation.ChangeSetTypeCreate
	if exists {
		operation = cloudformation.ChangeSetTypeUpdate
	}

	awsParams := make([]*cloudformation.Parameter, 0, len(params))

	for k, v := range params {
		awsParams = append(awsParams, &cloudformation.Parameter{
			ParameterKey:   aws.String(k),
			ParameterValue: aws.String(v),
		})
	}

	createOut, err := ap.cf.CreateChangeSet(&cloudformation.CreateChangeSetInput{
		ChangeSetType: aws.String(operation),
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
	return ap.cf.WaitUntilChangeSetCreateCompleteWithContext(
		aws.BackgroundContext(),
		&cloudformation.DescribeChangeSetInput{
			ChangeSetName: aws.String(ID),
		},
		func(w *request.Waiter) {
			w.Delay = request.ConstantWaiterDelay(time.Second)
		},
	)
}

// ChangeSetChanges returns info about changes to be applied to stack
func (ap *AwsProvider) ChangeSetChanges(ID string) ([]claws.Change, error) {
	setInfo, err := ap.cf.DescribeChangeSet(&cloudformation.DescribeChangeSetInput{
		ChangeSetName: aws.String(ID),
	})

	if err != nil {
		return []claws.Change{}, err
	}

	if *setInfo.Status == cloudformation.ChangeSetStatusFailed {
		if *setInfo.StatusReason == noChangeStatus {
			return []claws.Change{}, claws.ErrNoChange
		}
		return []claws.Change{}, errors.New(*setInfo.StatusReason)
	}

	changes := make([]claws.Change, 0, len(setInfo.Changes))

	for _, c := range setInfo.Changes {
		awsChange := c.ResourceChange
		ch := claws.Change{
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
func (ap *AwsProvider) StackEvents(stackName string) ([]claws.StackEvent, error) {
	awsEvents, err := ap.cf.DescribeStackEvents(&cloudformation.DescribeStackEventsInput{
		StackName: aws.String(stackName),
	})

	if err != nil {
		return []claws.StackEvent{}, err
	}

	events := make([]claws.StackEvent, len(awsEvents.StackEvents))

	for i, e := range awsEvents.StackEvents {
		events[i] = claws.StackEvent{
			ID:                *e.EventId,
			ResourceType:      fromAwsString(e.ResourceType),
			Status:            fromAwsString(e.ResourceStatus),
			LogicalResourceID: fromAwsString(e.LogicalResourceId),
			StatusReason:      fromAwsString(e.ResourceStatusReason),
		}
	}

	return events, nil
}

// BlockResource prevents a stack resource from deletion and replacement
func (ap *AwsProvider) BlockResource(stackName string, resource string) error {
	policy := `{
		"Statement" : [{
			"Effect" : "Deny",
			"Action" : [
				"Update:Replace",
				"Update:Delete"
			],
			"Principal": "*",
			"Resource" : "LogicalResourceId/%s"
		}]
	}`

	return ap.applyPolicy(stackName, fmt.Sprintf(policy, resource))
}

// UnblockResource discards the blocking from the resource
func (ap *AwsProvider) UnblockResource(stackName string, resource string) error {
	policy := `{
		"Statement" : [{
			"Effect" : "Allow",
			"Action" : "Update:*",
			"Principal": "*",
			"Resource" : "LogicalResourceId/%s"
		}]
	}`

	return ap.applyPolicy(stackName, fmt.Sprintf(policy, resource))
}

func (ap *AwsProvider) applyPolicy(stackName string, policy string) error {
	_, err := ap.cf.SetStackPolicy(&cloudformation.SetStackPolicyInput{
		StackName:       aws.String(stackName),
		StackPolicyBody: aws.String(policy),
	})
	return err
}

func fromAwsString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
