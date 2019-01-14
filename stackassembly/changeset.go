package stackassembly

import (
	"errors"
	"fmt"
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
	parameters map[string]string
	tags       map[string]string
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

func (cs *ChangeSet) Register() (*ChangeSetHandle, error) {
	chSet := &ChangeSetHandle{
		cf:        cs.stack.cf,
		stackName: cs.stack.Name,
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

	output, err := cs.stack.cf.CreateChangeSet(&cloudformation.CreateChangeSetInput{
		ChangeSetType: aws.String(operation),
		ChangeSetName: aws.String("chst-" + strconv.FormatInt(time.Now().UnixNano(), 10)),
		TemplateBody:  aws.String(cs.body),
		StackName:     aws.String(cs.stack.Name),
		Parameters:    awsParams,
		Tags:          cs.awsTags(),
		Capabilities:  []*string{aws.String("CAPABILITY_IAM")},
	})

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
	output, err := cs.stack.cf.ValidateTemplate(&cloudformation.ValidateTemplateInput{
		TemplateBody: aws.String(cs.body),
	})
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

	for k, v := range cs.tags {
		awsTags = append(awsTags, &cloudformation.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	return awsTags
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
