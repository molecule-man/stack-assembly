package stackassembly

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
)

const noChangeStatus = "The submitted information didn't contain changes. " +
	"Submit different information to create a change set."

// StackEvent is a stack event
type StackEvent struct {
	ID                string
	ResourceType      string
	Status            string
	LogicalResourceID string
	StatusReason      string
	Timestamp         time.Time
}

//ErrNoChange is error that indicate that there are no changes to apply
var ErrNoChange = errors.New("no changes")

var ErrStackDoesntExist = errors.New("stack doesn't exist")

type Stack struct {
	Name string

	cf         cloudformationiface.CloudFormationAPI
	cachedInfo *StackInfo
}

func NewStack(cf cloudformationiface.CloudFormationAPI, name string) *Stack {
	return &Stack{cf: cf, Name: name}
}

func (s *Stack) Info() (StackInfo, error) {
	if s.cachedInfo != nil {
		return *s.cachedInfo, nil
	}

	info := StackInfo{}

	stack, err := s.describe()
	if err != nil {
		return info, err
	}

	info.awsStack = stack
	s.cachedInfo = &info
	return info, err
}

func (s *Stack) describe() (*cloudformation.Stack, error) {
	info, err := s.cf.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(s.Name),
	})

	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			return nil, ErrStackDoesntExist
		}
		return nil, err
	}

	return info.Stacks[0], nil
}

func (s *Stack) Exists() (bool, error) {
	_, err := s.Info()

	if err == ErrStackDoesntExist {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func (s *Stack) Delete() error {
	_, err := s.cf.DeleteStack(&cloudformation.DeleteStackInput{
		StackName: aws.String(s.Name),
	})
	if err != nil {
		return err
	}

	waitInput := cloudformation.DescribeStacksInput{
		StackName: aws.String(s.Name),
	}
	ctx := aws.BackgroundContext()
	return s.cf.WaitUntilStackDeleteCompleteWithContext(ctx, &waitInput, func(w *request.Waiter) {
		w.MaxAttempts = 900
		w.Delay = request.ConstantWaiterDelay(2 * time.Second)
	})
}

func (s *Stack) AlreadyDeployed() (bool, error) {
	exists, err := s.Exists()
	if err != nil {
		return false, err
	}

	if !exists {
		return false, nil
	}

	info, err := s.Info()
	if err != nil {
		return false, err
	}

	return !info.InReviewState(), nil
}

func (s *Stack) ChangeSet(body string) *ChangeSet {
	return &ChangeSet{
		stack:      s,
		body:       body,
		parameters: map[string]string{},
	}
}

func (s *Stack) Body() (string, error) {
	tpl, err := s.cf.GetTemplate(&cloudformation.GetTemplateInput{
		StackName: aws.String(s.Name),
	})

	if err != nil {
		return "", err
	}

	return aws.StringValue(tpl.TemplateBody), nil
}

// Resources returns info about stack resources
func (s *Stack) Resources() ([]StackResource, error) {
	resp, err := s.cf.DescribeStackResources(&cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(s.Name),
	})

	if err != nil {
		return []StackResource{}, err
	}

	resources := make([]StackResource, len(resp.StackResources))

	for i, r := range resp.StackResources {
		resource := StackResource{
			LogicalID: *r.LogicalResourceId,
			Status:    *r.ResourceStatus,
			Type:      *r.ResourceType,
			Timestamp: *r.Timestamp,
		}

		if r.PhysicalResourceId != nil {
			resource.PhysicalID = *r.PhysicalResourceId
		}
		if r.ResourceStatusReason != nil {
			resource.StatusReason = *r.ResourceStatusReason
		}

		resources[i] = resource
	}

	return resources, nil
}

func (s *Stack) Events() ([]StackEvent, error) {
	awsEvents, err := s.cf.DescribeStackEvents(&cloudformation.DescribeStackEventsInput{
		StackName: aws.String(s.Name),
	})

	if err != nil {
		return []StackEvent{}, err
	}

	events := make([]StackEvent, len(awsEvents.StackEvents))

	for i, e := range awsEvents.StackEvents {
		events[i] = StackEvent{
			ID:                aws.StringValue(e.EventId),
			ResourceType:      aws.StringValue(e.ResourceType),
			Status:            aws.StringValue(e.ResourceStatus),
			LogicalResourceID: aws.StringValue(e.LogicalResourceId),
			StatusReason:      aws.StringValue(e.ResourceStatusReason),
			Timestamp:         aws.TimeValue(e.Timestamp),
		}
	}

	return events, nil
}

// BlockResource prevents a stack resource from deletion and replacement
func (s *Stack) BlockResource(resource string) error {
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

	return s.applyPolicy(fmt.Sprintf(policy, resource))
}

// UnblockResource discards the blocking from the resource
func (s *Stack) UnblockResource(resource string) error {
	policy := `{
		"Statement" : [{
			"Effect" : "Allow",
			"Action" : "Update:*",
			"Principal": "*",
			"Resource" : "LogicalResourceId/%s"
		}]
	}`

	return s.applyPolicy(fmt.Sprintf(policy, resource))
}

func (s *Stack) applyPolicy(policy string) error {
	_, err := s.cf.SetStackPolicy(&cloudformation.SetStackPolicyInput{
		StackName:       aws.String(s.Name),
		StackPolicyBody: aws.String(policy),
	})
	return err
}
