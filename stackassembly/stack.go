package stackassembly

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
)

const noChangeStatus = "The submitted information didn't contain changes. " +
	"Submit different information to create a change set."

type StackConfig struct {
	Name       string
	Path       string
	Body       string
	Parameters map[string]string
	Tags       map[string]string
	DependsOn  []string
	Blocked    []string
}

type Stack struct {
	Name    string
	Blocked []string

	parameters map[string]string
	tags       map[string]string
	body       string
	path       string

	exists bool

	cf cloudformationiface.CloudFormationAPI
}

// Change is a change that is applied to the stack
type Change struct {
	Action            string
	ResourceType      string
	LogicalResourceID string
	ReplacementNeeded bool
}

// StackEvent is a stack event
type StackEvent struct {
	ID                string
	ResourceType      string
	Status            string
	LogicalResourceID string
	StatusReason      string
	Timestamp         time.Time
}

// StackOutput contains info about stack output variables
type StackOutput struct {
	Key         string
	Value       string
	Description string
	ExportName  string
}

type StackResource struct {
	LogicalID    string
	PhysicalID   string
	Status       string
	StatusReason string
	Type         string
	Timestamp    time.Time
}

type StackDetails struct {
	Name       string
	Body       string
	Parameters []KeyVal
	Tags       []KeyVal
}

type StackStatus struct {
	Status            string
	StatusDescription string
}

type KeyVal struct {
	Key string
	Val string
}

//ErrNoChange is error that indicate that there are no changes to apply
var ErrNoChange = errors.New("no changes")

var ErrStackDoesntExist = errors.New("stack doesn't exist")

func NewStack(cf cloudformationiface.CloudFormationAPI, stackCfg StackConfig, globalParameters map[string]string) (Stack, error) {
	stack := Stack{}

	stack.cf = cf

	stack.path = stackCfg.Path

	stack.Blocked = stackCfg.Blocked
	stack.tags = stackCfg.Tags

	stack.parameters = make(map[string]string, len(globalParameters)+len(stackCfg.Parameters))

	for k, v := range globalParameters {
		if _, ok := stackCfg.Parameters[k]; !ok {
			stack.parameters[k] = v
		}
	}

	data := struct{ Params map[string]string }{}
	data.Params = stack.parameters

	for k, v := range stackCfg.Parameters {
		var parsed string
		if err := applyTemplating(&parsed, v, data); err != nil {
			return stack, err
		}
		stack.parameters[k] = parsed
	}

	data.Params = stack.parameters

	stack.tags = make(map[string]string, len(stackCfg.Tags))
	for k, v := range stackCfg.Tags {
		var parsed string
		if err := applyTemplating(&parsed, v, data); err != nil {
			return stack, err
		}
		stack.tags[k] = parsed
	}

	err := applyTemplating(&stack.Name, stackCfg.Name, data)

	return stack, err
}

func (s *Stack) Body() (string, error) {
	if s.body != "" {
		return s.body, nil
	}
	tplBody, err := ioutil.ReadFile(s.path)
	if err != nil {
		return "", err
	}

	data := struct{ Params map[string]string }{}
	data.Params = s.parameters
	err = applyTemplating(&s.body, string(tplBody), data)
	return s.body, err
}

type ChangeSetHandle struct {
	ID        string
	Changes   []Change
	stackName string
	exists    bool
	cf        cloudformationiface.CloudFormationAPI
}

func (csh ChangeSetHandle) Exec() error {
	_, err := csh.cf.ExecuteChangeSet(&cloudformation.ExecuteChangeSetInput{
		ChangeSetName: aws.String(csh.ID),
	})
	if err != nil {
		return err
	}

	stackInput := cloudformation.DescribeStacksInput{
		StackName: aws.String(csh.stackName),
	}

	ctx := aws.BackgroundContext()

	if csh.exists {
		return csh.cf.WaitUntilStackUpdateCompleteWithContext(ctx, &stackInput, func(w *request.Waiter) {
			w.MaxAttempts = 900
			w.Delay = request.ConstantWaiterDelay(3 * time.Second)
		})
	}

	return csh.cf.WaitUntilStackCreateCompleteWithContext(ctx, &stackInput, func(w *request.Waiter) {
		w.MaxAttempts = 900
		w.Delay = request.ConstantWaiterDelay(3 * time.Second)
	})
}

func (s *Stack) ChangeSet() (ChangeSetHandle, error) {
	chSet := ChangeSetHandle{
		cf:        s.cf,
		stackName: s.Name,
	}

	body, err := s.Body()
	if err != nil {
		return chSet, err
	}

	awsParams, err := s.awsParameters()
	if err != nil {
		return chSet, err
	}

	exists, err := s.Exists()
	if err != nil {
		return chSet, err
	}

	chSet.exists = exists

	operation := cloudformation.ChangeSetTypeCreate
	if exists {
		operation = cloudformation.ChangeSetTypeUpdate
	}

	createOut, err := s.cf.CreateChangeSet(&cloudformation.CreateChangeSetInput{
		ChangeSetType: aws.String(operation),
		ChangeSetName: aws.String("chst-" + strconv.FormatInt(time.Now().UnixNano(), 10)),
		TemplateBody:  aws.String(body),
		StackName:     aws.String(s.Name),
		Parameters:    awsParams,
		Tags:          s.awsTags(),
		Capabilities:  []*string{aws.String("CAPABILITY_IAM")},
	})

	if err != nil {
		return chSet, err
	}

	chSet.ID = aws.StringValue(createOut.Id)

	err = s.waitChangeSet(createOut.Id)
	if err != nil {
		return chSet, err
	}

	chSet.Changes, err = s.ChangeSetChanges(createOut.Id)
	if err != nil {
		return chSet, err
	}

	return chSet, err
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

	for _, stack := range info.Stacks {
		if aws.StringValue(stack.StackStatus) == cloudformation.StackStatusReviewInProgress {
			return nil, ErrStackDoesntExist
		}
	}

	return info.Stacks[0], nil
}

func (s *Stack) awsParameters() ([]*cloudformation.Parameter, error) {
	body, err := s.Body()
	if err != nil {
		return []*cloudformation.Parameter{}, err
	}

	out, err := s.cf.ValidateTemplate(&cloudformation.ValidateTemplateInput{
		TemplateBody: aws.String(body),
	})
	if err != nil {
		return []*cloudformation.Parameter{}, err
	}

	awsParams := make([]*cloudformation.Parameter, 0, len(out.Parameters))

	for _, p := range out.Parameters {
		k := aws.StringValue(p.ParameterKey)
		if v, ok := s.parameters[k]; ok {
			awsParams = append(awsParams, &cloudformation.Parameter{
				ParameterKey:   aws.String(k),
				ParameterValue: aws.String(v),
			})
		}
	}

	return awsParams, nil
}

func (s *Stack) awsTags() []*cloudformation.Tag {
	awsTags := make([]*cloudformation.Tag, 0, len(s.tags))

	for k, v := range s.tags {
		awsTags = append(awsTags, &cloudformation.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	return awsTags
}

func (s *Stack) waitChangeSet(id *string) error {
	err := s.cf.WaitUntilChangeSetCreateCompleteWithContext(
		aws.BackgroundContext(),
		&cloudformation.DescribeChangeSetInput{
			ChangeSetName: id,
		},
		func(w *request.Waiter) {
			w.Delay = request.ConstantWaiterDelay(3 * time.Second)
		},
	)

	if aerr, ok := err.(awserr.Error); ok {
		if aerr.Code() == request.WaiterResourceNotReadyErrorCode {
			setInfo, derr := s.cf.DescribeChangeSet(&cloudformation.DescribeChangeSetInput{
				ChangeSetName: id,
			})

			if derr != nil {
				return fmt.Errorf("error while retrieving more info about change set failure: %v", derr)
			}

			if aws.StringValue(setInfo.StatusReason) == "No updates are to be performed." {
				return ErrNoChange
			}

			if aws.StringValue(setInfo.StatusReason) == noChangeStatus {
				return ErrNoChange
			}

			return fmt.Errorf("[%s] %s. Status: %s, StatusReason: %s", *setInfo.ChangeSetId, err.Error(), *setInfo.Status, *setInfo.StatusReason)
		}
	}
	return err
}

func (s *Stack) Status() (StackStatus, error) {
	status := StackStatus{}
	stack, err := s.describe()
	if err != nil {
		return status, err
	}

	status.Status = aws.StringValue(stack.StackStatus)
	status.StatusDescription = aws.StringValue(stack.StackStatusReason)
	return status, nil
}

func (s *Stack) Details() (StackDetails, error) {
	details := StackDetails{}

	tpl, err := s.cf.GetTemplate(&cloudformation.GetTemplateInput{
		StackName: aws.String(s.Name),
	})

	if err != nil {
		return details, err
	}

	details.Body = *tpl.TemplateBody

	stack, err := s.describe()
	if err != nil {
		return details, err
	}

	details.Name = aws.StringValue(stack.StackName)

	details.Parameters = make([]KeyVal, 0, len(stack.Parameters))

	for _, p := range stack.Parameters {
		details.Parameters = append(details.Parameters, KeyVal{
			Key: aws.StringValue(p.ParameterKey),
			Val: aws.StringValue(p.ParameterValue),
		})
	}

	details.Tags = make([]KeyVal, 0, len(stack.Tags))

	for _, t := range stack.Tags {
		details.Tags = append(details.Tags, KeyVal{
			Key: aws.StringValue(t.Key),
			Val: aws.StringValue(t.Value),
		})
	}

	return details, nil
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

func (s *Stack) Exists() (bool, error) {
	if s.exists {
		return true, nil
	}

	_, err := s.describe()

	if err == ErrStackDoesntExist {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	s.exists = true
	return true, nil
}

func (s *Stack) ChangeSetChanges(ID *string) ([]Change, error) {
	changes := make([]Change, 0)
	return changes, s.changes(ID, &changes, nil)
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

// Outputs returns the "outputs" of a stack
func (s *Stack) Outputs() ([]StackOutput, error) {
	outputs := make([]StackOutput, 0)

	resp, err := s.cf.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(s.Name),
	})

	if err != nil {
		return outputs, err
	}

	for _, s := range resp.Stacks {
		outputs = make([]StackOutput, len(s.Outputs))
		for i, o := range s.Outputs {
			out := StackOutput{
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

func (s *Stack) changes(ID *string, store *[]Change, nextToken *string) error {
	setInfo, err := s.cf.DescribeChangeSet(&cloudformation.DescribeChangeSetInput{
		ChangeSetName: ID,
		NextToken:     nextToken,
	})

	if err != nil {
		return err
	}

	if *setInfo.Status == cloudformation.ChangeSetStatusFailed {

		if *setInfo.StatusReason == noChangeStatus {
			return ErrNoChange
		}
		return errors.New(*setInfo.StatusReason)
	}

	for _, c := range setInfo.Changes {
		awsChange := c.ResourceChange
		ch := Change{
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
		return s.changes(ID, store, setInfo.NextToken)
	}

	return nil
}

func applyTemplating(parsed *string, tpl string, data interface{}) error {
	t, err := template.New(tpl).Parse(tpl)
	if err != nil {
		return err
	}
	var buff bytes.Buffer

	if err := t.Execute(&buff, data); err != nil {
		return err
	}

	*parsed = buff.String()
	return nil
}
