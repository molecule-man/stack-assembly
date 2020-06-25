package mock

import (
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	clf "github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/molecule-man/stack-assembly/aws"
)

type WriteProvider struct {
	testID     string
	featureID  string
	scenarioID string
}

func (p WriteProvider) Must(cfg aws.Config) *aws.AWS {
	a, err := p.New(cfg)

	if err != nil {
		panic(err)
	}

	return a
}

func (p WriteProvider) New(cfg aws.Config) (*aws.AWS, error) {
	realP := aws.Provider{}

	raws, err := realP.New(cfg)
	if err != nil {
		return raws, err
	}

	d := newDumper(p.testID, p.featureID, p.scenarioID)
	d.addReplacement(raws.AccountID, "AWS_ACC_ID")
	d.addReplacement(raws.Region, "AWS_REGION")

	cf := &CloudFormation{realCF: raws.CF, dumper: d}
	s3 := &S3UploadManagerWriteProvider{realMGR: raws.S3UploadManager, dumper: d}

	return &aws.AWS{
		CF:              cf,
		S3UploadManager: s3,
		S3:              raws.S3,
		AccountID:       raws.AccountID,
		Region:          raws.Region,
	}, nil
}

type CloudFormation struct {
	cloudformationiface.CloudFormationAPI
	realCF cloudformationiface.CloudFormationAPI
	dumper *dumper
}

func (c *CloudFormation) DescribeStacks(input *clf.DescribeStacksInput) (*clf.DescribeStacksOutput, error) {
	output, err := c.realCF.DescribeStacks(input)
	c.dumper.dump("DescribeStacks", input, output, err)

	return output, err
}

func (c *CloudFormation) ValidateTemplate(input *clf.ValidateTemplateInput) (*clf.ValidateTemplateOutput, error) {
	output, err := c.realCF.ValidateTemplate(input)
	c.dumper.dump("ValidateTemplate", input, output, err)

	return output, err
}

func (c *CloudFormation) CreateChangeSet(input *clf.CreateChangeSetInput) (*clf.CreateChangeSetOutput, error) {
	output, err := c.realCF.CreateChangeSet(input)

	c.dumper.addReplacement(awssdk.StringValue(input.ChangeSetName), "CHST_ID")
	c.dumper.dump("CreateChangeSet", input, output, err)

	return output, err
}

func (c *CloudFormation) DescribeChangeSet(input *clf.DescribeChangeSetInput) (*clf.DescribeChangeSetOutput, error) {
	output, err := c.realCF.DescribeChangeSet(input)
	c.dumper.dump("DescribeChangeSet", input, output, err)

	return output, err
}

func (c *CloudFormation) DescribeStackEvents(input *clf.DescribeStackEventsInput) (*clf.DescribeStackEventsOutput, error) {
	output, err := c.realCF.DescribeStackEvents(input)
	c.dumper.dump("DescribeStackEvents", input, output, err)

	return output, err
}

func (c *CloudFormation) DescribeStackResource(input *clf.DescribeStackResourceInput) (*clf.DescribeStackResourceOutput, error) {
	output, err := c.realCF.DescribeStackResource(input)
	c.dumper.dump("DescribeStackResource", input, output, err)

	return output, err
}

func (c *CloudFormation) DescribeStackResources(input *clf.DescribeStackResourcesInput) (*clf.DescribeStackResourcesOutput, error) {
	output, err := c.realCF.DescribeStackResources(input)
	c.dumper.dump("DescribeStackResources", input, output, err)

	return output, err
}

func (c *CloudFormation) ExecuteChangeSet(input *clf.ExecuteChangeSetInput) (*clf.ExecuteChangeSetOutput, error) {
	output, err := c.realCF.ExecuteChangeSet(input)
	c.dumper.dump("ExecuteChangeSet", input, output, err)

	return output, err
}

func (c *CloudFormation) DeleteStack(input *clf.DeleteStackInput) (*clf.DeleteStackOutput, error) {
	output, err := c.realCF.DeleteStack(input)
	c.dumper.dump("DeleteStack", input, output, err)

	return output, err
}

func (c *CloudFormation) WaitUntilChangeSetCreateCompleteWithContext(
	ctx awssdk.Context,
	input *clf.DescribeChangeSetInput,
	opts ...request.WaiterOption,
) error {
	err := c.realCF.WaitUntilChangeSetCreateCompleteWithContext(ctx, input, opts...)
	c.dumper.dump("WaitUntilChangeSetCreateCompleteWithContext", input, "", err)

	return err
}

func (c *CloudFormation) WaitUntilStackCreateCompleteWithContext(
	ctx awssdk.Context,
	input *clf.DescribeStacksInput,
	opts ...request.WaiterOption,
) error {
	err := c.realCF.WaitUntilStackCreateCompleteWithContext(ctx, input, opts...)
	c.dumper.dump("WaitUntilStackCreateCompleteWithContext", input, "", err)

	return err
}

func (c *CloudFormation) WaitUntilStackUpdateCompleteWithContext(
	ctx awssdk.Context,
	input *clf.DescribeStacksInput,
	opts ...request.WaiterOption,
) error {
	err := c.realCF.WaitUntilStackUpdateCompleteWithContext(ctx, input, opts...)
	c.dumper.dump("WaitUntilStackUpdateCompleteWithContext", input, "", err)

	return err
}

func (c *CloudFormation) WaitUntilStackDeleteCompleteWithContext(
	ctx awssdk.Context,
	input *clf.DescribeStacksInput,
	opts ...request.WaiterOption,
) error {
	err := c.realCF.WaitUntilStackDeleteCompleteWithContext(ctx, input, opts...)
	c.dumper.dump("WaitUntilStackDeleteCompleteWithContext", input, "", err)

	return err
}

func (c *CloudFormation) SetStackPolicy(input *clf.SetStackPolicyInput) (*clf.SetStackPolicyOutput, error) {
	output, err := c.realCF.SetStackPolicy(input)
	c.dumper.dump("SetStackPolicy", input, output, err)

	return output, err
}

func (c *CloudFormation) GetTemplate(input *clf.GetTemplateInput) (*clf.GetTemplateOutput, error) {
	output, err := c.realCF.GetTemplate(input)
	c.dumper.dump("GetTemplate", input, output, err)

	return output, err
}

type S3UploadManagerWriteProvider struct {
	realMGR aws.S3UploadManager
	dumper  *dumper
}

func (w *S3UploadManagerWriteProvider) Upload(
	input *s3manager.UploadInput,
	opts ...func(*s3manager.Uploader),
) (*s3manager.UploadOutput, error) {
	output, err := w.realMGR.Upload(input, opts...)
	w.dumper.dump("s3uploader_Upload", input, output, err)

	return output, err
}
