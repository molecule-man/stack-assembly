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
	"github.com/molecule-man/stack-assembly/depgraph"
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
	ID         string
	Name       string
	Parameters map[string]string
	Tags       map[string]string
	DependsOn  []string
	Blocked    []string

	body string
	path string

	Cf cloudformationiface.CloudFormationAPI
}

func NewStack(id string, stackCfg StackConfig, globalParameters map[string]string) (Stack, error) {
	stack := Stack{}

	stack.path = stackCfg.Path

	stack.ID = id
	stack.DependsOn = stackCfg.DependsOn
	stack.Blocked = stackCfg.Blocked
	stack.Tags = stackCfg.Tags

	stack.Parameters = make(map[string]string, len(globalParameters)+len(stackCfg.Parameters))

	for k, v := range globalParameters {
		if _, ok := stackCfg.Parameters[k]; !ok {
			stack.Parameters[k] = v
		}
	}

	data := struct{ Params map[string]string }{}
	data.Params = stack.Parameters

	for k, v := range stackCfg.Parameters {
		var parsed string
		if err := applyTemplating(&parsed, v, data); err != nil {
			return stack, err
		}
		stack.Parameters[k] = parsed
	}

	data.Params = stack.Parameters

	stack.Tags = make(map[string]string, len(stackCfg.Tags))
	for k, v := range stackCfg.Tags {
		var parsed string
		if err := applyTemplating(&parsed, v, data); err != nil {
			return stack, err
		}
		stack.Tags[k] = parsed
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
	data.Params = s.Parameters
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
		cf:        s.Cf,
		stackName: s.Name,
	}

	body, err := s.Body()
	if err != nil {
		return chSet, err
	}

	awsParams, err := s.buildAwsParams(body)
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

	createOut, err := s.Cf.CreateChangeSet(&cloudformation.CreateChangeSetInput{
		ChangeSetType: aws.String(operation),
		ChangeSetName: aws.String("chst-" + strconv.FormatInt(time.Now().UnixNano(), 10)),
		TemplateBody:  aws.String(body),
		StackName:     aws.String(s.Name),
		Parameters:    awsParams,
		Tags:          s.buildAwsTags(),
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

func (s *Stack) buildAwsParams(body string) ([]*cloudformation.Parameter, error) {
	out, err := s.Cf.ValidateTemplate(&cloudformation.ValidateTemplateInput{
		TemplateBody: aws.String(body),
	})
	if err != nil {
		return []*cloudformation.Parameter{}, err
	}

	awsParams := make([]*cloudformation.Parameter, 0, len(out.Parameters))

	for _, p := range out.Parameters {
		k := aws.StringValue(p.ParameterKey)
		if v, ok := s.Parameters[k]; ok {
			awsParams = append(awsParams, &cloudformation.Parameter{
				ParameterKey:   aws.String(k),
				ParameterValue: aws.String(v),
			})
		}
	}

	return awsParams, nil
}
func (s *Stack) buildAwsTags() []*cloudformation.Tag {
	awsTags := make([]*cloudformation.Tag, 0, len(s.Tags))

	for k, v := range s.Tags {
		awsTags = append(awsTags, &cloudformation.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	return awsTags
}

func (s *Stack) waitChangeSet(id *string) error {
	err := s.Cf.WaitUntilChangeSetCreateCompleteWithContext(
		aws.BackgroundContext(),
		&cloudformation.DescribeChangeSetInput{
			ChangeSetName: id,
		},
		func(w *request.Waiter) {
			w.Delay = request.ConstantWaiterDelay(2 * time.Second)
		},
	)

	if aerr, ok := err.(awserr.Error); ok {
		if aerr.Code() == request.WaiterResourceNotReadyErrorCode {
			setInfo, derr := s.Cf.DescribeChangeSet(&cloudformation.DescribeChangeSetInput{
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
	awsEvents, err := s.Cf.DescribeStackEvents(&cloudformation.DescribeStackEventsInput{
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
	info, err := s.Cf.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(s.Name),
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

func (s *Stack) ChangeSetChanges(ID *string) ([]Change, error) {
	changes := make([]Change, 0)
	return changes, s.changes(ID, &changes, nil)
}

func (s *Stack) changes(ID *string, store *[]Change, nextToken *string) error {
	setInfo, err := s.Cf.DescribeChangeSet(&cloudformation.DescribeChangeSetInput{
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

func SortStacksByExecOrder(stacks []Stack) error {
	dg := depgraph.DepGraph{}

	stacksMap := make(map[string]Stack, len(stacks))

	for _, stack := range stacks {
		dg.Add(stack.ID, stack.DependsOn)
		stacksMap[stack.ID] = stack
	}

	orderedIds, err := dg.Resolve()
	if err != nil {
		return err
	}

	for i, id := range orderedIds {
		stacks[i] = stacksMap[id]
	}

	return nil
}
