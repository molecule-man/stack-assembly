package mock

import (
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	clf "github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
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
	d := newDumper(p.testID, p.featureID, p.scenarioID)

	return &aws.AWS{
		CF:              &GfCloudFormation{dumper: d},
		S3UploadManager: &S3UploadManagerReadProvider{dumper: d},
		AccountID:       "ACCID",
		Region:          "eu-west-1",
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

func (c *GfCloudFormation) WaitUntilChangeSetCreateCompleteWithContext(
	_ awssdk.Context,
	input *clf.DescribeChangeSetInput,
	opts ...request.WaiterOption,
) error {
	return c.dumper.read("WaitUntilChangeSetCreateCompleteWithContext", input, "")
}

func (c *GfCloudFormation) WaitUntilStackCreateCompleteWithContext(
	_ awssdk.Context,
	input *clf.DescribeStacksInput,
	_ ...request.WaiterOption,
) error {
	return c.dumper.read("WaitUntilStackCreateCompleteWithContext", input, "")
}

func (c *GfCloudFormation) WaitUntilStackUpdateCompleteWithContext(
	_ awssdk.Context,
	input *clf.DescribeStacksInput,
	_ ...request.WaiterOption,
) error {
	return c.dumper.read("WaitUntilStackUpdateCompleteWithContext", input, "")
}

func (c *GfCloudFormation) WaitUntilStackDeleteCompleteWithContext(
	_ awssdk.Context,
	input *clf.DescribeStacksInput,
	_ ...request.WaiterOption,
) error {
	return c.dumper.read("WaitUntilStackDeleteCompleteWithContext", input, "")
}

func (c *GfCloudFormation) DeleteStack(input *clf.DeleteStackInput) (*clf.DeleteStackOutput, error) {
	output := &clf.DeleteStackOutput{}
	return output, c.dumper.read("DeleteStack", input, output)
}

func (c *GfCloudFormation) SetStackPolicy(input *clf.SetStackPolicyInput) (*clf.SetStackPolicyOutput, error) {
	output := &clf.SetStackPolicyOutput{}
	return output, c.dumper.read("SetStackPolicy", input, output)
}

func (c *GfCloudFormation) GetTemplate(input *clf.GetTemplateInput) (*clf.GetTemplateOutput, error) {
	output := &clf.GetTemplateOutput{}
	return output, c.dumper.read("GetTemplate", input, output)
}

type S3UploadManagerReadProvider struct {
	aws.S3UploadManager
	dumper *dumper
}

func (r *S3UploadManagerReadProvider) Upload(
	input *s3manager.UploadInput,
	opts ...func(*s3manager.Uploader),
) (*s3manager.UploadOutput, error) {
	output := &s3manager.UploadOutput{}
	return output, r.dumper.read("s3uploader_Upload", input, output)
}
