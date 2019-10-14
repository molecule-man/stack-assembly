package mock

import (
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	clf "github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/molecule-man/stack-assembly/aws"
)

type ReadProvider struct {
	testID     string
	featureID  string
	scenarioID string
}

func (p ReadProvider) Must(cfg aws.Config) *aws.AWS {
	a, err := p.New(cfg)

	if err != nil {
		panic(err)
	}

	return a
}

func (p ReadProvider) New(cfg aws.Config) (*aws.AWS, error) {
	return &aws.AWS{
		CF: &GfCloudFormation{
			dumper: newDumper(p.testID, p.featureID, p.scenarioID),
		},
		AccountID: "ACCID",
		Region:    "eu-west-1",
	}, nil
}

type GfCloudFormation struct {
	cloudformationiface.CloudFormationAPI
	dumper *dumper
}

func (c *GfCloudFormation) DescribeStacks(input *clf.DescribeStacksInput) (*clf.DescribeStacksOutput, error) {
	output := &clf.DescribeStacksOutput{}
	err := c.dumper.read("DescribeStacks", input, output)
	return output, err
}

func (c *GfCloudFormation) ValidateTemplate(input *clf.ValidateTemplateInput) (*clf.ValidateTemplateOutput, error) {
	output := &clf.ValidateTemplateOutput{}
	err := c.dumper.read("ValidateTemplate", input, output)
	return output, err
}

func (c *GfCloudFormation) CreateChangeSet(input *clf.CreateChangeSetInput) (*clf.CreateChangeSetOutput, error) {
	c.dumper.addReplacement(awssdk.StringValue(input.ChangeSetName), "CHST_ID")

	output := &clf.CreateChangeSetOutput{}
	err := c.dumper.read("CreateChangeSet", input, output)
	return output, err
}

func (c *GfCloudFormation) DescribeChangeSet(input *clf.DescribeChangeSetInput) (*clf.DescribeChangeSetOutput, error) {
	output := &clf.DescribeChangeSetOutput{}
	err := c.dumper.read("DescribeChangeSet", input, output)
	return output, err
}

func (c *GfCloudFormation) DescribeStackEvents(input *clf.DescribeStackEventsInput) (*clf.DescribeStackEventsOutput, error) {
	output := &clf.DescribeStackEventsOutput{}
	err := c.dumper.read("DescribeStackEvents", input, output)
	return output, err
}

func (c *GfCloudFormation) DescribeStackResource(input *clf.DescribeStackResourceInput) (*clf.DescribeStackResourceOutput, error) {
	output := &clf.DescribeStackResourceOutput{}
	err := c.dumper.read("DescribeStackResource", input, output)
	return output, err
}

func (c *GfCloudFormation) DescribeStackResources(input *clf.DescribeStackResourcesInput) (*clf.DescribeStackResourcesOutput, error) {
	output := &clf.DescribeStackResourcesOutput{}
	err := c.dumper.read("DescribeStackResources", input, output)
	return output, err
}

func (c *GfCloudFormation) ExecuteChangeSet(input *clf.ExecuteChangeSetInput) (*clf.ExecuteChangeSetOutput, error) {
	output := &clf.ExecuteChangeSetOutput{}
	err := c.dumper.read("ExecuteChangeSet", input, output)
	return output, err
}

func (c *GfCloudFormation) WaitUntilChangeSetCreateCompleteWithContext(_ awssdk.Context, input *clf.DescribeChangeSetInput, opts ...request.WaiterOption) error {
	err := c.dumper.read("WaitUntilChangeSetCreateCompleteWithContext", input, "")
	return err
}

func (c *GfCloudFormation) WaitUntilStackCreateCompleteWithContext(_ awssdk.Context, input *clf.DescribeStacksInput, _ ...request.WaiterOption) error {
	err := c.dumper.read("WaitUntilStackCreateCompleteWithContext", input, "")
	return err
}

func (c *GfCloudFormation) WaitUntilStackUpdateCompleteWithContext(_ awssdk.Context, input *clf.DescribeStacksInput, _ ...request.WaiterOption) error {
	err := c.dumper.read("WaitUntilStackUpdateCompleteWithContext", input, "")
	return err
}

func (c *GfCloudFormation) WaitUntilStackDeleteCompleteWithContext(_ awssdk.Context, input *clf.DescribeStacksInput, _ ...request.WaiterOption) error {
	err := c.dumper.read("WaitUntilStackDeleteCompleteWithContext", input, "")
	return err
}

func (c *GfCloudFormation) DeleteStack(input *clf.DeleteStackInput) (*clf.DeleteStackOutput, error) {
	output := &clf.DeleteStackOutput{}
	err := c.dumper.read("DeleteStack", input, output)
	return output, err
}

func (c *GfCloudFormation) SetStackPolicy(input *clf.SetStackPolicyInput) (*clf.SetStackPolicyOutput, error) {
	output := &clf.SetStackPolicyOutput{}
	err := c.dumper.read("SetStackPolicy", input, output)
	return output, err
}

func (c *GfCloudFormation) GetTemplate(input *clf.GetTemplateInput) (*clf.GetTemplateOutput, error) {
	output := &clf.GetTemplateOutput{}
	err := c.dumper.read("GetTemplate", input, output)
	return output, err
}
