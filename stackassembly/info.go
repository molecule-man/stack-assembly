package stackassembly

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

type KeyVal struct {
	Key string
	Val string
}

type StackInfo struct {
	awsStack *cloudformation.Stack
	exists   bool
}

func (si StackInfo) Exists() bool {
	return si.exists
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
