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

// StackEvent is a stack event
type StackEvent struct {
	ID                string
	ResourceType      string
	Status            string
	LogicalResourceID string
	StatusReason      string
	Timestamp         time.Time
}

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

	cf cloudformationiface.CloudFormationAPI
}

// Change is a change that is applied to the stack
type Change struct {
	Action            string
	ResourceType      string
	LogicalResourceID string
	ReplacementNeeded bool
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

	isUpdate := true

	info, err := s.Info()
	switch {
	case err == ErrStackDoesntExist:
		isUpdate = false
	case err != nil:
		return chSet, err
	case info.InReviewState():
		isUpdate = false
	}

	chSet.isUpdate = isUpdate

	operation := cloudformation.ChangeSetTypeCreate
	if isUpdate {
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

func (s *Stack) Info() (StackInfo, error) {
	stack, err := s.describe()
	info := StackInfo{}
	info.awsStack = stack
	info.exists = err != ErrStackDoesntExist
	info.cf = s.cf
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

func (s *Stack) ChangeSetChanges(ID *string) ([]Change, error) {
	changes := make([]Change, 0)
	return changes, s.changes(ID, &changes, nil)
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
