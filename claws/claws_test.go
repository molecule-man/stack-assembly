package claws

import (
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/molecule-man/claws/cloudprov"
)

func TestEventLog(t *testing.T) {

	cp := &CloudProviderMock{
		events: []cloudprov.StackEvent{{ID: "3"}, {ID: "2"}, {ID: "1"}},
		waitStackFunc: func() error {
			time.Sleep(10 * time.Millisecond)
			return nil
		},
	}

	var logs []string

	cs, _ := New(
		cp,
		StackTemplate{StackName: "stack", Body: "body", Params: map[string]string{}},
		WithEventSleep(6*time.Millisecond),
		WithEventSubscriber(func(e cloudprov.StackEvent) {
			cp.Lock()
			logs = append(logs, "log"+e.ID)
			cp.Unlock()
		}),
	)

	go func() {
		for _, i := range []string{"4", "5", "6"} {
			time.Sleep(1 * time.Millisecond)
			cp.Lock()
			cp.events = append([]cloudprov.StackEvent{{ID: i}}, cp.events...)
			cp.Unlock()
		}
	}()
	cs.Exec()

	logged := strings.Join(logs, " ")
	expected := "log4 log5 log6"

	if logged != expected {
		t.Errorf("Expected to be logged \"%s\". Actually logged \"%s\"", expected, logged)
	}
}

func TestOperationIsCreateIfStackDoesntExist(t *testing.T) {
	cp := &CloudProviderMock{}
	cp.stackExists = false
	New(cp, StackTemplate{})

	if cp.operation != cloudprov.CreateOperation {
		t.Error("If stack doesn't exist, changeset should be created with 'Create' operation")
	}
}

func TestOperationIsUpdateIfStackExists(t *testing.T) {
	cp := &CloudProviderMock{}
	cp.stackExists = true
	New(cp, StackTemplate{})

	if cp.operation != cloudprov.UpdateOperation {
		t.Error("If stack exists, changeset should be created with 'Update' operation")
	}
}

func TestOnlyRequiredParametersAreSubmitted(t *testing.T) {
	cp := &CloudProviderMock{}
	cp.requiredParams = []string{"foo", "bar"}
	New(cp, StackTemplate{Params: map[string]string{
		"foo": "fooval",
		"bar": "barval",
		"buz": "buzval",
	}})

	expected := map[string]string{"foo": "fooval", "bar": "barval"}
	if !reflect.DeepEqual(expected, cp.submittedParams) {
		t.Errorf("Expected params %v to be submitted. Got %v", expected, cp.submittedParams)
	}
}

func TestChangeSetCreationErrors(t *testing.T) {
	cases := []struct {
		errProv func(*CloudProviderMock, error)
		err     error
	}{
		{func(cp *CloudProviderMock, err error) { cp.validationErr = err }, errors.New("invalid")},
		{func(cp *CloudProviderMock, err error) { cp.stackExistsErr = err }, errors.New("stackExistsErr")},
		{func(cp *CloudProviderMock, err error) { cp.createErr = err }, errors.New("createErr")},
		{func(cp *CloudProviderMock, err error) { cp.changesErr = err }, errors.New("changesErr")},
		{func(cp *CloudProviderMock, err error) { cp.waitChSetErr = err }, errors.New("waitChSetErr")},
	}
	for _, tc := range cases {
		cp := &CloudProviderMock{}
		tc.errProv(cp, tc.err)

		_, err := New(cp, StackTemplate{})

		if tc.err != err {
			t.Errorf("Expected to get error %v. Got %v", tc.err, err)
		}

	}
}
func TestChangeSetExecutionErrors(t *testing.T) {
	cases := []struct {
		errProv func(*CloudProviderMock, error)
		err     error
	}{
		{func(cp *CloudProviderMock, err error) { cp.execErr = err }, errors.New("execErr")},
		{func(cp *CloudProviderMock, err error) { cp.waitStackErr = err }, errors.New("waitStackErr")},
	}
	for _, tc := range cases {
		cp := &CloudProviderMock{}
		tc.errProv(cp, tc.err)

		cs, _ := New(cp, StackTemplate{})
		err := cs.Exec()

		if tc.err != err {
			t.Errorf("Expected to get error %v. Got %v", tc.err, err)
		}

	}
}

type CloudProviderMock struct {
	sync.Mutex
	waitStackFunc   func() error
	events          []cloudprov.StackEvent
	chSetID         string
	operation       cloudprov.ChangeSetOperation
	stackExists     bool
	requiredParams  []string
	submittedParams map[string]string
	validationErr   error
	stackExistsErr  error
	createErr       error
	changesErr      error
	waitChSetErr    error
	execErr         error
	waitStackErr    error
}

func (cpm *CloudProviderMock) ValidateTemplate(tplBody string) ([]string, error) {
	return cpm.requiredParams, cpm.validationErr
}
func (cpm *CloudProviderMock) StackExists(stackName string) (bool, error) {
	return cpm.stackExists, cpm.stackExistsErr
}
func (cpm *CloudProviderMock) CreateChangeSet(stackName string, tplBody string, params map[string]string, op cloudprov.ChangeSetOperation) (string, error) {
	cpm.operation = op
	cpm.submittedParams = params

	if cpm.chSetID == "" {
		return "ID", cpm.createErr
	}
	return cpm.chSetID, cpm.createErr
}
func (cpm *CloudProviderMock) WaitChangeSetCreated(ID string) error {
	return cpm.waitChSetErr
}
func (cpm *CloudProviderMock) ChangeSetChanges(ID string) ([]cloudprov.Change, error) {
	return []cloudprov.Change{}, cpm.changesErr
}
func (cpm *CloudProviderMock) ExecuteChangeSet(ID string) error {
	return cpm.execErr
}
func (cpm *CloudProviderMock) WaitStack(stackName string) error {
	if cpm.waitStackFunc != nil {
		return cpm.waitStackFunc()
	}
	return cpm.waitStackErr
}
func (cpm *CloudProviderMock) StackEvents(stackName string) ([]cloudprov.StackEvent, error) {
	cpm.Lock()
	defer cpm.Unlock()
	return cpm.events, nil
}
