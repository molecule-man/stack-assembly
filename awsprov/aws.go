package awsprov

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/molecule-man/stack-assembly/stackassembly"
)

const noChangeStatus = "The submitted information didn't contain changes. " +
	"Submit different information to create a change set."

// AwsProvider is wrapper over aws sdk
type AwsProvider struct {
	cf *cloudformation.CloudFormation
}

// Config of AwsProvider
type Config struct {
	Profile string
	Region  string
}

// New creates a new AwsProvider
func New(c Config) *AwsProvider {
	opts := session.Options{
		Profile: c.Profile,
	}

	cfg := aws.Config{}
	if c.Region != "" {
		cfg.Region = aws.String(c.Region)
	}
	httpClient := http.Client{
		Timeout: 2 * time.Second,
	}
	cfg.HTTPClient = &httpClient

	opts.Config = cfg

	sess := session.Must(session.NewSessionWithOptions(opts))
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

	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			return false, nil
		}
		return false, err
	}

	for _, stack := range info.Stacks {
		if *stack.StackStatus == cloudformation.StackStatusReviewInProgress {
			return false, nil
		}
	}

	return true, nil
}

func (ap *AwsProvider) StackDetails(stackName string) (stackassembly.StackDetails, error) {
	details := stackassembly.StackDetails{}

	tpl, err := ap.cf.GetTemplate(&cloudformation.GetTemplateInput{
		StackName: aws.String(stackName),
	})

	if err != nil {
		return details, err
	}

	details.Body = *tpl.TemplateBody

	return details, nil
}

// StackOutputs returns the "outputs" of a stack identified by stackName
func (ap *AwsProvider) StackOutputs(stackName string) ([]stackassembly.StackOutput, error) {
	outputs := make([]stackassembly.StackOutput, 0)

	resp, err := ap.cf.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})

	if err != nil {
		return outputs, err
	}

	for _, s := range resp.Stacks {
		outputs = make([]stackassembly.StackOutput, len(s.Outputs))
		for i, o := range s.Outputs {
			out := stackassembly.StackOutput{
				Key:   *o.OutputKey,
				Value: *o.OutputValue,
			}

			if o.Description != nil {
				out.Description = *o.Description
			}

			if o.ExportName != nil {
				out.ExportName = *o.ExportName
			}

			outputs[i] = out
		}
	}

	return outputs, nil
}

// StackResources returns info about stack resources
func (ap *AwsProvider) StackResources(stackName string) ([]stackassembly.StackResource, error) {
	resp, err := ap.cf.DescribeStackResources(&cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(stackName),
	})

	if err != nil {
		return []stackassembly.StackResource{}, err
	}

	resources := make([]stackassembly.StackResource, len(resp.StackResources))

	for i, r := range resp.StackResources {
		resource := stackassembly.StackResource{
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

// CreateChangeSet creates new change set
// Returns change set ID
func (ap *AwsProvider) CreateChangeSet(stackName string, tplBody string, params, tags map[string]string) (string, error) {
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

	awsTags := make([]*cloudformation.Tag, 0, len(tags))

	for k, v := range tags {
		awsTags = append(awsTags, &cloudformation.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	createOut, err := ap.cf.CreateChangeSet(&cloudformation.CreateChangeSetInput{
		ChangeSetType: aws.String(operation),
		ChangeSetName: aws.String("chst-" + strconv.FormatInt(time.Now().UnixNano(), 10)),
		TemplateBody:  aws.String(tplBody),
		StackName:     aws.String(stackName),
		Parameters:    awsParams,
		Tags:          awsTags,
		Capabilities:  []*string{aws.String("CAPABILITY_IAM")},
	})

	if err != nil {
		return "", err
	}

	return *createOut.Id, nil
}

// WaitChangeSetCreated blocks runtime until the change set is not created
func (ap *AwsProvider) WaitChangeSetCreated(ID string) error {
	err := ap.cf.WaitUntilChangeSetCreateCompleteWithContext(
		aws.BackgroundContext(),
		&cloudformation.DescribeChangeSetInput{
			ChangeSetName: aws.String(ID),
		},
		func(w *request.Waiter) {
			w.Delay = request.ConstantWaiterDelay(time.Second)
		},
	)

	if err == nil {
		return nil
	}

	if aerr, ok := err.(awserr.Error); ok {
		if aerr.Code() == request.WaiterResourceNotReadyErrorCode {
			setInfo, derr := ap.cf.DescribeChangeSet(&cloudformation.DescribeChangeSetInput{
				ChangeSetName: aws.String(ID),
			})

			if derr != nil {
				return fmt.Errorf("error while retrieving more info about change set failure: %v", derr)
			}

			return fmt.Errorf("[%s] %s. Status: %s, StatusReason: %s", *setInfo.ChangeSetId, err.Error(), *setInfo.Status, *setInfo.StatusReason)
		}
	}
	return err
}

// ChangeSetChanges returns info about changes to be applied to stack
func (ap *AwsProvider) ChangeSetChanges(ID string) ([]stackassembly.Change, error) {
	changes := make([]stackassembly.Change, 0)
	return changes, ap.changes(ID, &changes, nil)
}

func (ap *AwsProvider) changes(ID string, store *[]stackassembly.Change, nextToken *string) error {
	setInfo, err := ap.cf.DescribeChangeSet(&cloudformation.DescribeChangeSetInput{
		ChangeSetName: aws.String(ID),
		NextToken:     nextToken,
	})

	if err != nil {
		return err
	}

	if *setInfo.Status == cloudformation.ChangeSetStatusFailed {
		if *setInfo.StatusReason == noChangeStatus {
			return stackassembly.ErrNoChange
		}
		return errors.New(*setInfo.StatusReason)
	}

	for _, c := range setInfo.Changes {
		awsChange := c.ResourceChange
		ch := stackassembly.Change{
			Action:            *awsChange.Action,
			ResourceType:      *awsChange.ResourceType,
			LogicalResourceID: *awsChange.LogicalResourceId,
		}
		if awsChange.Replacement != nil && *awsChange.Replacement == "True" {
			ch.ReplacementNeeded = true
		}

		*store = append(*store, ch)
	}

	if setInfo.NextToken != nil && *setInfo.NextToken != "" {
		return ap.changes(ID, store, setInfo.NextToken)
	}

	return nil
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

	c := make(chan error)
	ctx, cancel := context.WithCancel(aws.BackgroundContext())

	go func() {
		c <- ap.cf.WaitUntilStackUpdateCompleteWithContext(ctx, &stackInput, func(w *request.Waiter) {
			w.MaxAttempts = 900
			w.Delay = request.ConstantWaiterDelay(2 * time.Second)
		})
	}()

	go func() {
		c <- ap.cf.WaitUntilStackCreateCompleteWithContext(ctx, &stackInput, func(w *request.Waiter) {
			w.MaxAttempts = 900
			w.Delay = request.ConstantWaiterDelay(2 * time.Second)
		})
	}()

	err := <-c
	cancel()

	return err
}

// StackEvents returns stack events. The latest events appear first
func (ap *AwsProvider) StackEvents(stackName string) ([]stackassembly.StackEvent, error) {
	awsEvents, err := ap.cf.DescribeStackEvents(&cloudformation.DescribeStackEventsInput{
		StackName: aws.String(stackName),
	})

	if err != nil {
		return []stackassembly.StackEvent{}, err
	}

	events := make([]stackassembly.StackEvent, len(awsEvents.StackEvents))

	for i, e := range awsEvents.StackEvents {
		events[i] = stackassembly.StackEvent{
			ID:                *e.EventId,
			ResourceType:      fromAwsString(e.ResourceType),
			Status:            fromAwsString(e.ResourceStatus),
			LogicalResourceID: fromAwsString(e.LogicalResourceId),
			StatusReason:      fromAwsString(e.ResourceStatusReason),
			Timestamp:         *e.Timestamp,
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
