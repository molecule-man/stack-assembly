package awscf

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
)

type ChangeSet struct {
	stack      *Stack
	body       string
	url        string
	parameters map[string]string
	tags       map[string]string

	input cloudformation.CreateChangeSetInput
}

// Change is a change that is applied to the stack
type Change struct {
	Action            string
	ResourceType      string
	LogicalResourceID string
	ReplacementNeeded bool
}

func (cs *ChangeSet) Stack() *Stack {
	return cs.stack
}

func (cs *ChangeSet) WithTemplateURL(url string) *ChangeSet {
	if url != "" {
		cs.url = url
	}

	return cs
}

func (cs *ChangeSet) WithParameters(parameters map[string]string) *ChangeSet {
	cs.parameters = parameters
	return cs
}

func (cs *ChangeSet) WithParameter(k, v string) *ChangeSet {
	cs.parameters[k] = v
	return cs
}

func (cs *ChangeSet) WithTags(tags map[string]string) *ChangeSet {
	cs.tags = tags
	return cs
}

func (cs *ChangeSet) WithRollback(rollbackCfg *cloudformation.RollbackConfiguration) *ChangeSet {
	cs.input.RollbackConfiguration = rollbackCfg
	return cs
}

func (cs *ChangeSet) WithClientToken(token string) *ChangeSet {
	if token != "" {
		cs.input.ClientToken = aws.String(token)
	}

	return cs
}

func (cs *ChangeSet) WithCapabilities(capabilities []string) *ChangeSet {
	cs.input.Capabilities = aws.StringSlice(capabilities)
	return cs
}

func (cs *ChangeSet) WithResourceTypes(types []string) *ChangeSet {
	if len(types) > 0 {
		cs.input.ResourceTypes = aws.StringSlice(types)
	}

	return cs
}

func (cs *ChangeSet) WithRoleARN(arn string) *ChangeSet {
	if arn != "" {
		cs.input.RoleARN = aws.String(arn)
	}

	return cs
}

func (cs *ChangeSet) WithUsePrevTpl(usePrevTpl bool) *ChangeSet {
	if usePrevTpl {
		cs.input.UsePreviousTemplate = aws.Bool(true)
	}

	return cs
}

func (cs *ChangeSet) WithNotificationARNs(arns []string) *ChangeSet {
	if len(arns) > 0 {
		cs.input.NotificationARNs = aws.StringSlice(arns)
	}

	return cs
}

func (cs *ChangeSet) Register() (*ChangeSetHandle, error) {
	chSet := &ChangeSetHandle{
		cf:        cs.stack.cf,
		stackName: cs.stack.Name,
	}

	if err := cs.setupTplLocation(); err != nil {
		return chSet, err
	}

	awsParams, err := cs.awsParameters()
	if err != nil {
		return chSet, err
	}

	chSet.IsUpdate, err = cs.stack.AlreadyDeployed()
	if err != nil {
		return chSet, err
	}

	operation := cloudformation.ChangeSetTypeCreate
	if chSet.IsUpdate {
		operation = cloudformation.ChangeSetTypeUpdate
	}

	cs.input.ChangeSetType = aws.String(operation)
	cs.input.ChangeSetName = aws.String("chst-" + strconv.FormatInt(time.Now().UnixNano(), 10))
	cs.input.StackName = aws.String(cs.stack.Name)
	cs.input.Parameters = awsParams
	cs.input.Tags = cs.awsTags()

	output, err := cs.stack.cf.CreateChangeSet(&cs.input)

	if err != nil {
		return chSet, err
	}

	chSet.ID = aws.StringValue(output.Id)

	err = cs.wait(output.Id)
	if err != nil {
		return chSet, err
	}

	return chSet, chSet.loadChanges()
}

func (cs *ChangeSet) setupTplLocation() error {
	if cs.url != "" {
		cs.input.TemplateURL = aws.String(cs.url)
		return nil
	}

	url, err := cs.stack.uploader.Upload(cs.body)
	if err != nil {
		return err
	}

	if url != "" {
		cs.input.TemplateURL = aws.String(url)
		return nil
	}

	cs.input.TemplateBody = aws.String(cs.body)

	return nil
}

func (cs *ChangeSet) wait(id *string) error {
	err := cs.stack.cf.WaitUntilChangeSetCreateCompleteWithContext(
		aws.BackgroundContext(),
		&cloudformation.DescribeChangeSetInput{
			ChangeSetName: id,
		},
		func(w *request.Waiter) {
			w.Delay = request.ConstantWaiterDelay(1 * time.Second)
		},
	)

	if aerr, ok := err.(awserr.Error); ok {
		if aerr.Code() == request.WaiterResourceNotReadyErrorCode {
			setInfo, derr := cs.stack.cf.DescribeChangeSet(&cloudformation.DescribeChangeSetInput{
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

func (cs *ChangeSet) awsParameters() ([]*cloudformation.Parameter, error) {
	input := cloudformation.ValidateTemplateInput{}

	switch {
	case cs.input.TemplateURL != nil:
		input.TemplateURL = cs.input.TemplateURL
	case cs.input.TemplateBody != nil:
		input.TemplateBody = cs.input.TemplateBody
	default:
		input.TemplateBody = &cs.body
	}

	output, err := cs.stack.cf.ValidateTemplate(&input)
	if err != nil {
		return []*cloudformation.Parameter{}, err
	}

	pb := newChSetParams(cs.stack, cs.parameters)

	for _, p := range output.Parameters {
		pb.add(p)
	}

	return pb.collect()
}

func (cs *ChangeSet) awsTags() []*cloudformation.Tag {
	awsTags := make([]*cloudformation.Tag, 0, len(cs.tags))

	keys := make([]string, 0, len(cs.tags))
	for k := range cs.tags {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		awsTags = append(awsTags, &cloudformation.Tag{
			Key:   aws.String(k),
			Value: aws.String(cs.tags[k]),
		})
	}

	return awsTags
}

func (cs *ChangeSet) Close() error {
	return cs.stack.uploader.Cleanup()
}

type ChangeSetHandle struct {
	ID        string
	Changes   []Change
	IsUpdate  bool
	stackName string
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

	if csh.IsUpdate {
		return csh.cf.WaitUntilStackUpdateCompleteWithContext(ctx, &stackInput, func(w *request.Waiter) {
			w.MaxAttempts = 900
			w.Delay = request.ConstantWaiterDelay(2 * time.Second)
		})
	}

	return csh.cf.WaitUntilStackCreateCompleteWithContext(ctx, &stackInput, func(w *request.Waiter) {
		w.MaxAttempts = 900
		w.Delay = request.ConstantWaiterDelay(2 * time.Second)
	})
}

func (csh *ChangeSetHandle) loadChanges() error {
	csh.Changes = make([]Change, 0)
	return csh.changes(&csh.Changes, nil)
}

func (csh ChangeSetHandle) changes(store *[]Change, nextToken *string) error {
	setInfo, err := csh.cf.DescribeChangeSet(&cloudformation.DescribeChangeSetInput{
		ChangeSetName: aws.String(csh.ID),
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
			Action:            aws.StringValue(awsChange.Action),
			ResourceType:      aws.StringValue(awsChange.ResourceType),
			LogicalResourceID: aws.StringValue(awsChange.LogicalResourceId),
		}

		if aws.StringValue(awsChange.Replacement) == "True" {
			ch.ReplacementNeeded = true
		}

		*store = append(*store, ch)
	}

	if aws.StringValue(setInfo.NextToken) != "" {
		return csh.changes(store, setInfo.NextToken)
	}

	return nil
}
