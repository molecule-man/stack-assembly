package stackassembly

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangeSetExecutedIfApproved(t *testing.T) {
	cp := &cpMock{}
	s := Service{
		Approver:      &FakedApprover{approved: true},
		Log:           &FakedLogger{},
		CloudProvider: cp,
	}

	err := s.Sync(makeConfigFromOneTemplate(StackConfig{}))

	require.NoError(t, err)
	assert.True(t, cp.executed, "It was expected that change set is executed")
}

func TestChangeSetIsCancelledIfNotApproved(t *testing.T) {
	cp := &cpMock{}
	s := Service{
		Approver:      &FakedApprover{approved: false},
		Log:           &FakedLogger{},
		CloudProvider: cp,
	}

	err := s.Sync(makeConfigFromOneTemplate(StackConfig{}))

	assert.False(t, cp.executed, "It was expected that change set is not executed")
	assert.Error(t, err)
}

func TestErrorIsReturnedIfChangeSetFails(t *testing.T) {
	expectedErr := errors.New("create err")
	cp := &cpMock{createErr: expectedErr}

	s := Service{Approver: &FakedApprover{approved: true}, Log: &FakedLogger{}, CloudProvider: cp}

	err := s.Sync(makeConfigFromOneTemplate(StackConfig{}))

	assert.False(t, cp.executed, "It was expected that change set is not executed")
	assert.EqualError(t, err, expectedErr.Error())
}

func TestSyncIsSuccessfullyIgnoredIfNoChanges(t *testing.T) {
	cp := &cpMock{createErr: ErrNoChange}

	s := Service{Approver: &FakedApprover{approved: true}, Log: &FakedLogger{}, CloudProvider: cp}

	err := s.Sync(makeConfigFromOneTemplate(StackConfig{}))

	require.NoError(t, err)
	assert.False(t, cp.executed, "It was expected that change set is not executed")
}

func TestExecErrorIsReturnedIfExecutionFails(t *testing.T) {
	expectedErr := errors.New("exec err")
	cp := &cpMock{execErr: expectedErr}

	s := Service{Approver: &FakedApprover{approved: true}, Log: &FakedLogger{}, CloudProvider: cp}

	err := s.Sync(makeConfigFromOneTemplate(StackConfig{}))
	assert.EqualError(t, err, expectedErr.Error())
}

func TestGlobalParametersAreMerged(t *testing.T) {
	cp := &cpMock{
		requiredParams: []string{"foo", "bar"},
	}
	s := Service{Approver: &FakedApprover{approved: true}, Log: &FakedLogger{}, CloudProvider: cp}

	err := s.Sync(Config{
		Stacks:     map[string]StackConfig{"stack1": {Parameters: map[string]string{"foo": "stack_foo"}}},
		Parameters: map[string]string{"bar": "global_bar", "buz": "global_buz"},
	})
	require.NoError(t, err)

	expected := map[string]string{"foo": "stack_foo", "bar": "global_bar"}
	assert.Equal(t, expected, cp.submittedParams)
}

func TestParametersCanBeTemplated(t *testing.T) {
	cp := &cpMock{
		requiredParams: []string{"serviceName"},
	}
	s := Service{Approver: &FakedApprover{approved: true}, Log: &FakedLogger{}, CloudProvider: cp}

	err := s.Sync(Config{
		Stacks: map[string]StackConfig{"stack1": {
			Parameters: map[string]string{"serviceName": "{{ .Params.name }}-{{ .Params.env }}"},
			Name:       "stack-{{ .Params.serviceName }}",
			Body:       "body: {{ .Params.serviceName }}-{{ .Params.foo }}",
		}},
		Parameters: map[string]string{"name": "acme", "env": "live", "foo": "bar"},
	})
	require.NoError(t, err)

	assert.Equal(t, map[string]string{"serviceName": "acme-live"}, cp.submittedParams)
	assert.Equal(t, "stack-acme-live", cp.name)
	assert.Equal(t, "body: acme-live-bar", cp.body)
}

func TestBlocking(t *testing.T) {
	cp := &cpMock{}
	s := Service{
		Approver:      &FakedApprover{approved: true},
		Log:           &FakedLogger{},
		CloudProvider: cp,
	}

	err := s.Sync(makeConfigFromOneTemplate(StackConfig{
		Blocked: []string{"foo", "bar"},
	}))

	require.NoError(t, err)

	cp.AssertBlocked(t, []string{"foo", "bar"})
}

func TestBlockingNoChange(t *testing.T) {
	cp := &cpMock{createErr: ErrNoChange}

	s := Service{
		Approver:      &FakedApprover{approved: true},
		Log:           &FakedLogger{},
		CloudProvider: cp,
	}

	err := s.Sync(makeConfigFromOneTemplate(StackConfig{
		Blocked: []string{"foo"},
	}))
	require.NoError(t, err)

	if err != nil {
		t.Errorf("It was expected that Sync is successful. Error %v was returned", err)
	}

	cp.AssertBlocked(t, []string{"foo"})
}

func makeConfigFromOneTemplate(stackCfg StackConfig) Config {
	return Config{
		Stacks: map[string]StackConfig{"stack": stackCfg},
	}
}

type FakedApprover struct {
	approved bool
}

func (fa *FakedApprover) Approve(c []Change) bool {
	return fa.approved
}

type FakedLogger struct{}

func (fl *FakedLogger) Print(s ...interface{}) {}
