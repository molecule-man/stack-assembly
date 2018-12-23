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

	err := s.Sync(StackTemplate{})

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

	err := s.Sync(StackTemplate{})

	assert.False(t, cp.executed, "It was expected that change set is not executed")
	assert.Error(t, err)
}

func TestErrorIsReturnedIfChangeSetFails(t *testing.T) {
	expectedErr := errors.New("create err")
	cp := &cpMock{createErr: expectedErr}

	s := Service{Approver: &FakedApprover{approved: true}, Log: &FakedLogger{}, CloudProvider: cp}

	err := s.Sync(StackTemplate{})

	assert.False(t, cp.executed, "It was expected that change set is not executed")
	assert.EqualError(t, err, expectedErr.Error())
}

func TestSyncIsSuccessfullyIgnoredIfNoChanges(t *testing.T) {
	cp := &cpMock{createErr: ErrNoChange}

	s := Service{Approver: &FakedApprover{approved: true}, Log: &FakedLogger{}, CloudProvider: cp}

	err := s.Sync(StackTemplate{})

	require.NoError(t, err)
	assert.False(t, cp.executed, "It was expected that change set is not executed")
}

func TestExecErrorIsReturnedIfExecutionFails(t *testing.T) {
	expectedErr := errors.New("exec err")
	cp := &cpMock{execErr: expectedErr}

	s := Service{Approver: &FakedApprover{approved: true}, Log: &FakedLogger{}, CloudProvider: cp}

	err := s.Sync(StackTemplate{})
	assert.EqualError(t, err, expectedErr.Error())
}

func TestGlobalParametersAreMerged(t *testing.T) {
	cp := &cpMock{
		requiredParams: []string{"foo", "bar"},
	}
	s := Service{Approver: &FakedApprover{approved: true}, Log: &FakedLogger{}, CloudProvider: cp}

	err := s.SyncAll(
		map[string]StackTemplate{"tpl1": {Parameters: map[string]string{"foo": "tpl_foo"}}},
		map[string]string{"bar": "global_bar", "buz": "global_buz"},
	)
	require.NoError(t, err)

	expected := map[string]string{"foo": "tpl_foo", "bar": "global_bar"}
	assert.Equal(t, expected, cp.submittedParams)
}

func TestParametersCanBeTemplated(t *testing.T) {
	cp := &cpMock{
		requiredParams: []string{"serviceName"},
	}
	s := Service{Approver: &FakedApprover{approved: true}, Log: &FakedLogger{}, CloudProvider: cp}

	err := s.SyncAll(
		map[string]StackTemplate{"tpl1": {
			Parameters: map[string]string{"serviceName": "{{ .Params.name }}-{{ .Params.env }}"},
			Name:       "stack-{{ .Params.serviceName }}",
			Body:       "body: {{ .Params.serviceName }}-{{ .Params.foo }}",
		}},
		map[string]string{"name": "acme", "env": "live", "foo": "bar"},
	)
	require.NoError(t, err)

	assert.Equal(t, map[string]string{"serviceName": "acme-live"}, cp.submittedParams)
	assert.Equal(t, "stack-acme-live", cp.name)
	assert.Equal(t, "body: acme-live-bar", cp.body)
}

func TestStackOutputsCanBeUsedInTemplating(t *testing.T) {
	cp := &cpMock{
		outputs: []StackOutput{{Key: "foo", Value: "bar"}},
	}
	s := Service{Approver: &FakedApprover{approved: true}, Log: &FakedLogger{}, CloudProvider: cp}

	err := s.SyncAll(
		map[string]StackTemplate{
			"tpl1": {
				Name:       "stack-{{ .Outputs.tpl2.foo }}-{{ .Params.buz}}",
				Parameters: map[string]string{"buz": "blah"},
			},
			"tpl2": {Name: "hello"},
		},
		map[string]string{},
	)
	require.NoError(t, err)
	assert.Equal(t, "stack-bar-blah", cp.name)
}

func TestBlocking(t *testing.T) {
	cp := &cpMock{}
	s := Service{
		Approver:      &FakedApprover{approved: true},
		Log:           &FakedLogger{},
		CloudProvider: cp,
	}

	err := s.Sync(StackTemplate{
		Blocked: []string{"foo", "bar"},
	})

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

	err := s.Sync(StackTemplate{
		Blocked: []string{"foo"},
	})
	require.NoError(t, err)

	if err != nil {
		t.Errorf("It was expected that Sync is successful. Error %v was returned", err)
	}

	cp.AssertBlocked(t, []string{"foo"})
}

type FakedApprover struct {
	approved bool
}

func (fa *FakedApprover) Approve(c []Change) bool {
	return fa.approved
}

type FakedLogger struct{}

func (fl *FakedLogger) Print(s ...interface{}) {}
