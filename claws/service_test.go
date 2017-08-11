package claws

import (
	"errors"
	"reflect"
	"testing"
)

func TestChangeSetExecutedIfApproved(t *testing.T) {
	cp := &CloudProviderMock{}
	s := Service{
		Approver:      &FakedApprover{approved: true},
		Log:           &FakedLogger{},
		CloudProvider: cp,
	}

	err := s.Sync(StackTemplate{})

	if !cp.executed {
		t.Error("It was expected that change set is executed")
	}
	if err != nil {
		t.Errorf("It was expected that Sync is successful. Error %v was returned", err)
	}
}

func TestChangeSetIsCancelledIfNotApproved(t *testing.T) {
	cp := &CloudProviderMock{}
	s := Service{
		Approver:      &FakedApprover{approved: false},
		Log:           &FakedLogger{},
		CloudProvider: cp,
	}

	err := s.Sync(StackTemplate{})

	if cp.executed {
		t.Error("It was expected that change set is not executed")
	}
	if err == nil {
		t.Error("It was expected that Sync returns error. Nil was returned")
	}
}

func TestErrorIsReturnedIfChangeSetFails(t *testing.T) {
	expectedErr := errors.New("create err")
	cp := &CloudProviderMock{createErr: expectedErr}

	s := Service{Approver: &FakedApprover{approved: true}, Log: &FakedLogger{}, CloudProvider: cp}

	err := s.Sync(StackTemplate{})

	if cp.executed {
		t.Error("It was expected that change set is not executed")
	}
	if err != expectedErr {
		t.Errorf("It was expected that Sync returns %v. %v was returned", expectedErr, err)
	}
}

func TestSyncIsSuccessfullyIgnoredIfNoChanges(t *testing.T) {
	cp := &CloudProviderMock{createErr: ErrNoChange}

	s := Service{Approver: &FakedApprover{approved: true}, Log: &FakedLogger{}, CloudProvider: cp}

	err := s.Sync(StackTemplate{})

	if cp.executed {
		t.Error("It was expected that change set is not executed")
	}
	if err != nil {
		t.Errorf("It was expected that Sync is successful. Error %v was returned", err)
	}
}

func TestExecErrorIsReturnedIfExecutionFails(t *testing.T) {
	expectedErr := errors.New("exec err")
	cp := &CloudProviderMock{execErr: expectedErr}

	s := Service{Approver: &FakedApprover{approved: true}, Log: &FakedLogger{}, CloudProvider: cp}

	err := s.Sync(StackTemplate{})

	if err != expectedErr {
		t.Errorf("It was expected that Sync returns %v. %v was returned", expectedErr, err)
	}
}

func TestGlobalParametersAreMerged(t *testing.T) {
	cp := &CloudProviderMock{
		requiredParams: []string{"foo", "bar"},
	}
	s := Service{Approver: &FakedApprover{approved: true}, Log: &FakedLogger{}, CloudProvider: cp}

	err := s.SyncAll(
		map[string]StackTemplate{"tpl1": {Params: map[string]string{"foo": "tpl_foo"}}},
		map[string]string{"bar": "global_bar", "buz": "global_buz"},
	)

	expected := map[string]string{"foo": "tpl_foo", "bar": "global_bar"}

	if !reflect.DeepEqual(expected, cp.submittedParams) {
		t.Errorf("Expected params %v to be submitted. Got %v", expected, cp.submittedParams)
	}
	if err != nil {
		t.Errorf("It was expected that SyncAll is successful. Error %v was returned", err)
	}
}
func TestParametersCanBeTemplated(t *testing.T) {
	cp := &CloudProviderMock{
		requiredParams: []string{"serviceName"},
	}
	s := Service{Approver: &FakedApprover{approved: true}, Log: &FakedLogger{}, CloudProvider: cp}

	err := s.SyncAll(
		map[string]StackTemplate{"tpl1": {
			Params: map[string]string{"serviceName": "{{ .name }}-{{ .env }}"},
			Name:   "stack-{{ .serviceName }}",
			Body:   "body: {{ .serviceName }}-{{ .foo }}",
		}},
		map[string]string{"name": "acme", "env": "live", "foo": "bar"},
	)

	expected := map[string]string{"serviceName": "acme-live"}

	if !reflect.DeepEqual(expected, cp.submittedParams) {
		t.Errorf("Expected params %v to be submitted. Got %v", expected, cp.submittedParams)
	}

	if expected := "stack-acme-live"; expected != cp.name {
		t.Errorf("Expected stack name: '%s'. Got '%s'", expected, cp.name)
	}

	if expected := "body: acme-live-bar"; expected != cp.body {
		t.Errorf("Expected stack body: '%s'. Got '%s'", expected, cp.body)
	}

	if err != nil {
		t.Errorf("It was expected that SyncAll is successful. Error %v was returned", err)
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
