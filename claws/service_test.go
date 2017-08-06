package claws

import (
	"errors"
	"testing"

	"github.com/molecule-man/claws/cloudprov"
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
	cp := &CloudProviderMock{createErr: cloudprov.ErrNoChange}

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

type FakedApprover struct {
	approved bool
}

func (fa *FakedApprover) Approve(c []cloudprov.Change) bool {
	return fa.approved
}

type FakedLogger struct{}

func (fl *FakedLogger) Print(s ...interface{}) {}
