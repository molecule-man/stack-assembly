package stackassembly

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
)

type KeyVal struct {
	Key string
	Val string
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

type StackInfo struct {
	awsStack *cloudformation.Stack
	err      error

	cf cloudformationiface.CloudFormationAPI
}

func (si StackInfo) Exists() bool {
	return si.err == nil
}

func (si StackInfo) AlreadyDeployed() bool {
	return si.Exists() && !si.InReviewState()
}

func (si StackInfo) Error() error {
	if si.err != nil && si.err != ErrStackDoesntExist {
		return si.err
	}

	return nil
}

func (si StackInfo) InReviewState() bool {
	return aws.StringValue(si.awsStack.StackStatus) == cloudformation.StackStatusReviewInProgress
}

func (si StackInfo) Status() string {
	return aws.StringValue(si.awsStack.StackStatus)
}

func (si StackInfo) StatusDescription() string {
	return aws.StringValue(si.awsStack.StackStatusReason)
}

func (si StackInfo) Parameters() []KeyVal {
	parameters := make([]KeyVal, 0, len(si.awsStack.Parameters))

	for _, p := range si.awsStack.Parameters {
		parameters = append(parameters, KeyVal{
			Key: aws.StringValue(p.ParameterKey),
			Val: aws.StringValue(p.ParameterValue),
		})
	}

	return parameters
}

func (si StackInfo) HasParameter(k string) bool {
	for _, p := range si.awsStack.Parameters {
		if aws.StringValue(p.ParameterKey) == k {
			return true
		}
	}
	return false
}

func (si StackInfo) Tags() []KeyVal {
	tags := make([]KeyVal, 0, len(si.awsStack.Tags))

	for _, t := range si.awsStack.Tags {
		tags = append(tags, KeyVal{
			Key: aws.StringValue(t.Key),
			Val: aws.StringValue(t.Value),
		})
	}

	return tags
}

// Outputs returns the "outputs" of a stack
func (si StackInfo) Outputs() []StackOutput {
	outputs := make([]StackOutput, len(si.awsStack.Outputs))
	for i, o := range si.awsStack.Outputs {
		out := StackOutput{
			Key:         aws.StringValue(o.OutputKey),
			Value:       aws.StringValue(o.OutputValue),
			Description: aws.StringValue(o.Description),
			ExportName:  aws.StringValue(o.ExportName),
		}

		outputs[i] = out
	}

	return outputs
}

func (si StackInfo) Body() (string, error) {
	tpl, err := si.cf.GetTemplate(&cloudformation.GetTemplateInput{
		StackName: si.awsStack.StackName,
	})

	if err != nil {
		return "", err
	}

	return aws.StringValue(tpl.TemplateBody), nil
}

// Resources returns info about stack resources
func (si StackInfo) Resources() ([]StackResource, error) {
	resp, err := si.cf.DescribeStackResources(&cloudformation.DescribeStackResourcesInput{
		StackName: si.awsStack.StackName,
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
