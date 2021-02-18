package awscf

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	saAws "github.com/molecule-man/stack-assembly/aws"
	"github.com/molecule-man/stack-assembly/errd"
)

const noChangeStatus = "The submitted information didn't contain changes. " +
	"Submit different information to create a change set."

// StackEvent is a stack event.
type StackEvent struct {
	ID                string
	ResourceType      string
	Status            string
	LogicalResourceID string
	StatusReason      string
	Timestamp         time.Time
}

type StackEvents []StackEvent

func (se StackEvents) Reversed() StackEvents {
	for i, j := 0, len(se)-1; i < j; i, j = i+1, j-1 {
		se[i], se[j] = se[j], se[i]
	}

	return se
}

// ErrNoChange is error that indicate that there are no changes to apply.
var ErrNoChange = errors.New("no changes")

var ErrStackDoesntExist = errors.New("stack doesn't exist")

var ErrStackAlreadyInProgress = errors.New("stack is already in progress")

type Stack struct {
	Name string

	cf          cloudformationiface.CloudFormationAPI
	uploader    *saAws.S3Uploader
	cachedInfo  *StackInfo
	eventsTrack *EventsTrack
}

func NewStack(name string, cf cloudformationiface.CloudFormationAPI, uploader *saAws.S3Uploader) *Stack {
	return &Stack{Name: name, cf: cf, uploader: uploader}
}

func (s *Stack) Info() (_ StackInfo, err error) {
	defer errd.Wrapf(&err, "failed to fetch stack info from aws")

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

func (s *Stack) Refresh() {
	s.cachedInfo = nil
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

func (s *Stack) getTemplateSummary() (*cloudformation.GetTemplateSummaryOutput, error) {
	summary, err := s.cf.GetTemplateSummary(&cloudformation.GetTemplateSummaryInput{
		StackName: aws.String(s.Name),
	})

	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			return nil, ErrStackDoesntExist
		}

		return nil, err
	}

	return summary, nil
}

func (s *Stack) Exists() (_ bool, err error) {
	defer errd.Wrapf(&err, "failed to check if stack exists")

	_, err = s.Info()

	if errors.Is(err, ErrStackDoesntExist) {
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

func (s *Stack) Wait() (err error) {
	defer errd.Wrapf(&err, "failed to wait until stack operation is completed")

	info, err := s.Info()
	if err != nil {
		return err
	}

	info.Status()

	ctx := aws.BackgroundContext()
	waitInput := cloudformation.DescribeStacksInput{
		StackName: aws.String(s.Name),
	}
	waiter := func(w *request.Waiter) {
		w.MaxAttempts = 900
		w.Delay = request.ConstantWaiterDelay(2 * time.Second)
	}

	if !strings.HasSuffix(info.Status(), "IN_PROGRESS") {
		return fmt.Errorf("stack is in state that can't be waited for: %s", info.Status())
	}

	switch {
	case strings.Contains(info.Status(), "ROLLBACK"):
		return s.cf.WaitUntilStackRollbackCompleteWithContext(ctx, &waitInput, waiter)
	case strings.Contains(info.Status(), "UPDATE"):
		return s.cf.WaitUntilStackUpdateCompleteWithContext(ctx, &waitInput, waiter)
	case strings.Contains(info.Status(), "CREATE"):
		return s.cf.WaitUntilStackCreateCompleteWithContext(ctx, &waitInput, waiter)
	case strings.Contains(info.Status(), "DELETE"):
		return s.cf.WaitUntilStackDeleteCompleteWithContext(ctx, &waitInput, waiter)
	}

	return fmt.Errorf("stack is in state that can't be waited for: %s", info.Status())
}

func (s *Stack) AlreadyDeployed() (_ bool, err error) {
	defer errd.Wrapf(&err, "failed to check if stack already deployed")

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

// Resources returns info about stack resources.
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

func (s *Stack) EventsTrack() *EventsTrack {
	if s.eventsTrack == nil {
		s.eventsTrack = &EventsTrack{stack: s}
	}

	return s.eventsTrack
}

// BlockResource prevents a stack resource from deletion and replacement.
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

// UnblockResource discards the blocking from the resource.
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
