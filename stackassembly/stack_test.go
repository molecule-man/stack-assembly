package stackassembly

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOnlyRequiredParametersAreSubmitted2(t *testing.T) {
	cf := &cfMock{}
	cf.templateParameters = []*cloudformation.TemplateParameter{
		{ParameterKey: aws.String("foo")},
		{ParameterKey: aws.String("bar")},
	}
	stack := Stack{cf: cf, body: "body", parameters: map[string]string{
		"foo": "fooval",
		"bar": "barval",
		"buz": "buzval",
	}}
	_, err := stack.ChangeSet()
	require.NoError(t, err)

	expected := []*cloudformation.Parameter{
		{ParameterKey: aws.String("foo"), ParameterValue: aws.String("fooval")},
		{ParameterKey: aws.String("bar"), ParameterValue: aws.String("barval")},
	}
	require.NotNil(t, cf.createChangeSetInput)
	assert.Equal(t, expected, cf.createChangeSetInput.Parameters)
}

func TestChangeSetCreationErrors2(t *testing.T) {
	cases := []struct {
		errProv func(*cfMock, error)
		err     error
	}{
		{func(cf *cfMock, err error) { cf.validationErr = err }, errors.New("invalid")},
		{func(cf *cfMock, err error) { cf.createErr = err }, errors.New("createErr")},
		{func(cf *cfMock, err error) { cf.changesErr = err }, errors.New("changesErr")},
		{func(cf *cfMock, err error) { cf.waitChSetErr = err }, errors.New("waitChSetErr")},
	}
	for _, tc := range cases {
		cf := &cfMock{}
		tc.errProv(cf, tc.err)

		stack := Stack{cf: cf, body: "body"}
		_, err := stack.ChangeSet()

		assert.EqualError(t, err, tc.err.Error())
	}
}

func TestEventTracking(t *testing.T) {
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}

	emmittedEvents := []*cloudformation.StackEvent{{EventId: aws.String("3")}, {EventId: aws.String("2")}, {EventId: aws.String("1")}}

	cf := &cfMock{
		waitStackFunc: func() error {
			wg.Wait()
			return nil
		},
		describeStackEventsFunc: func() (*cloudformation.DescribeStackEventsOutput, error) {
			mu.Lock()
			defer mu.Unlock()
			return &cloudformation.DescribeStackEventsOutput{
				StackEvents: emmittedEvents,
			}, nil
		},
	}

	capturedEvents := []StackEvent{}

	et := EventsTracker{
		sleep: 6 * time.Millisecond,
	}
	stack := Stack{cf: cf, body: "body"}
	cs, err := stack.ChangeSet()
	require.NoError(t, err)

	events, stop := et.StartTracking(&stack)

	wg.Add(1)
	go func() {
		for _, i := range []string{"4", "5", "6"} {
			time.Sleep(1 * time.Millisecond)
			mu.Lock()
			emmittedEvents = append([]*cloudformation.StackEvent{{EventId: aws.String(i)}}, emmittedEvents...)
			mu.Unlock()
		}
		wg.Done()
	}()

	require.NoError(t, cs.Exec())
	stop()
	for e := range events {
		capturedEvents = append(capturedEvents, e)
	}

	expected := []StackEvent{{ID: "4"}, {ID: "5"}, {ID: "6"}}

	assert.Equal(t, expected, capturedEvents)
}

type cfMock struct {
	cloudformationiface.CloudFormationAPI

	templateParameters []*cloudformation.TemplateParameter

	createChangeSetInput *cloudformation.CreateChangeSetInput

	err           error
	validationErr error
	createErr     error
	changesErr    error
	waitChSetErr  error
	describeErr   error

	body string

	waitStackFunc           func() error
	describeStackEventsFunc func() (*cloudformation.DescribeStackEventsOutput, error)
}

func (cf *cfMock) ValidateTemplate(*cloudformation.ValidateTemplateInput) (*cloudformation.ValidateTemplateOutput, error) {
	out := cloudformation.ValidateTemplateOutput{}
	out.Parameters = cf.templateParameters

	return &out, cf.validationErr
}

func (cf *cfMock) DescribeStacks(*cloudformation.DescribeStacksInput) (*cloudformation.DescribeStacksOutput, error) {
	out := cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{{}},
	}

	return &out, cf.describeErr
}
func (cf *cfMock) GetTemplate(*cloudformation.GetTemplateInput) (*cloudformation.GetTemplateOutput, error) {
	out := cloudformation.GetTemplateOutput{}
	out.TemplateBody = aws.String(cf.body)
	return &out, cf.err
}

func (cf *cfMock) DescribeChangeSet(*cloudformation.DescribeChangeSetInput) (*cloudformation.DescribeChangeSetOutput, error) {
	out := cloudformation.DescribeChangeSetOutput{}
	out.Status = aws.String("")

	return &out, cf.changesErr
}
func (cf *cfMock) DescribeStackEvents(input *cloudformation.DescribeStackEventsInput) (*cloudformation.DescribeStackEventsOutput, error) {
	if cf.describeStackEventsFunc != nil {
		return cf.describeStackEventsFunc()
	}
	return &cloudformation.DescribeStackEventsOutput{}, cf.err
}

func (cf *cfMock) CreateChangeSet(inp *cloudformation.CreateChangeSetInput) (*cloudformation.CreateChangeSetOutput, error) {
	cf.createChangeSetInput = inp
	out := cloudformation.CreateChangeSetOutput{}

	return &out, cf.createErr
}
func (cf *cfMock) ExecuteChangeSet(inp *cloudformation.ExecuteChangeSetInput) (*cloudformation.ExecuteChangeSetOutput, error) {
	return nil, cf.err
}

func (cf *cfMock) WaitUntilChangeSetCreateCompleteWithContext(
	aws.Context,
	*cloudformation.DescribeChangeSetInput,
	...request.WaiterOption) error {
	return cf.waitChSetErr
}
func (cf *cfMock) WaitUntilStackUpdateCompleteWithContext(aws.Context, *cloudformation.DescribeStacksInput, ...request.WaiterOption) error {
	if cf.waitStackFunc != nil {
		return cf.waitStackFunc()
	}
	return nil
}
