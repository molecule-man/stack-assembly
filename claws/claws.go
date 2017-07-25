package claws

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

//ErrNoChange is error
var ErrNoChange = errors.New("No changes")

type Claws struct {
	StackName string
	tplBody   string
	chSetID   *string
	operation string
	cf        *cloudformation.CloudFormation
}

// New is new
func New(stackName string, tplBody string) Claws {
	sess := session.Must(session.NewSession())

	return Claws{
		StackName: stackName,
		tplBody:   tplBody,
		operation: cloudformation.ChangeSetTypeUpdate,
		cf:        cloudformation.New(sess),
	}
}
func (c *Claws) NewChSet(userParams map[string]string) (*ChangeSet, error) {
	v := &cloudformation.ValidateTemplateInput{
		TemplateBody: aws.String(c.tplBody),
	}
	out, err := c.cf.ValidateTemplate(v)

	if err != nil {
		return nil, err
	}

	params := make(map[string]string, len(out.Parameters))

	for _, p := range out.Parameters {
		if v, ok := userParams[*p.ParameterKey]; ok {
			params[*p.ParameterKey] = v
		}
	}

	return c.createChangeSet(params)

}

func (c *Claws) createChangeSet(params map[string]string) (*ChangeSet, error) {
	info, err := c.cf.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(c.StackName),
	})

	if err == nil {
		for _, stack := range info.Stacks {
			if *stack.StackStatus == cloudformation.StackStatusReviewInProgress {
				c.operation = cloudformation.ChangeSetTypeCreate
			}
		}
	} else {
		if strings.Contains(err.Error(), "does not exist") {
			c.operation = cloudformation.ChangeSetTypeCreate
		} else {
			return nil, err
		}
	}

	awsParams := make([]*cloudformation.Parameter, 0, len(params))

	for k, v := range params {
		awsParams = append(awsParams, &cloudformation.Parameter{
			ParameterKey:   aws.String(k),
			ParameterValue: aws.String(v),
		})
	}

	createOut, err := c.cf.CreateChangeSet(&cloudformation.CreateChangeSetInput{
		ChangeSetType: aws.String(c.operation),
		ChangeSetName: aws.String("chst-" + strconv.FormatInt(time.Now().UnixNano(), 10)),
		TemplateBody:  aws.String(c.tplBody),
		StackName:     aws.String(c.StackName),
		Parameters:    awsParams,
	})

	if err != nil {
		return nil, err
	}

	c.chSetID = createOut.Id

	chSet := &ChangeSet{
		StackName: c.StackName,
		ID:        createOut.Id,
		operation: c.operation,
		cf:        c.cf,
	}

	if _, err = chSet.describe(); err != nil {
		return nil, err
	}

	err = c.cf.WaitUntilChangeSetCreateCompleteWithContext(aws.BackgroundContext(), &cloudformation.DescribeChangeSetInput{
		ChangeSetName: createOut.Id,
	}, func(w *request.Waiter) {
		w.Delay = request.ConstantWaiterDelay(time.Second)
	})

	if err != nil {
		return nil, err
	}

	chSetInfo, err := chSet.describe()

	if err != nil {
		return nil, err
	}
	chSet.Changes = extractChangesFromAwsResponse(chSetInfo.Changes)

	return chSet, nil
}

func extractChangesFromAwsResponse(awsChanges []*cloudformation.Change) []Change {
	changes := make([]Change, 0, len(awsChanges))
	for _, c := range awsChanges {
		awsChange := c.ResourceChange
		ch := Change{
			Action:            *awsChange.Action,
			ResourceType:      *awsChange.ResourceType,
			LogicalResourceId: *awsChange.LogicalResourceId,
		}
		if awsChange.Replacement != nil && *awsChange.Replacement == "True" {
			ch.ReplacementNeeded = true
		}

		changes = append(changes, ch)
	}

	return changes
}
