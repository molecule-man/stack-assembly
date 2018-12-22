package stackassembly

import (
	"errors"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

func assertNoError(t *testing.T, err error) {
	if err != nil {
		t.Errorf("It was not expected to get error %v", err)
	}
}

func TestEventLog(t *testing.T) {

	cp := &cpMock{
		events: []StackEvent{{ID: "3"}, {ID: "2"}, {ID: "1"}},
		waitStackFunc: func() error {
			time.Sleep(15 * time.Millisecond)
			return nil
		},
	}

	var logs []string

	cs, _ := New(
		cp,
		StackTemplate{Name: "stack", Body: "body", Params: map[string]string{}},
		WithEventSleep(6*time.Millisecond),
		WithEventSubscriber(func(e StackEvent) {
			cp.Lock()
			logs = append(logs, "log"+e.ID)
			cp.Unlock()
		}),
	)

	go func() {
		for _, i := range []string{"4", "5", "6"} {
			time.Sleep(1 * time.Millisecond)
			cp.Lock()
			cp.events = append([]StackEvent{{ID: i}}, cp.events...)
			cp.Unlock()
		}
	}()
	assertNoError(t, cs.Exec())

	logged := strings.Join(logs, " ")
	expected := "log4 log5 log6"

	if logged != expected {
		t.Errorf("Expected to be logged \"%s\". Actually logged \"%s\"", expected, logged)
	}
}

func TestOnlyRequiredParametersAreSubmitted(t *testing.T) {
	cp := &cpMock{}
	cp.requiredParams = []string{"foo", "bar"}
	_, err := New(cp, StackTemplate{Params: map[string]string{
		"foo": "fooval",
		"bar": "barval",
		"buz": "buzval",
	}})
	assertNoError(t, err)

	expected := map[string]string{"foo": "fooval", "bar": "barval"}
	if !reflect.DeepEqual(expected, cp.submittedParams) {
		t.Errorf("Expected params %v to be submitted. Got %v", expected, cp.submittedParams)
	}
}

func TestChangeSetCreationErrors(t *testing.T) {
	cases := []struct {
		errProv func(*cpMock, error)
		err     error
	}{
		{func(cp *cpMock, err error) { cp.validationErr = err }, errors.New("invalid")},
		{func(cp *cpMock, err error) { cp.createErr = err }, errors.New("createErr")},
		{func(cp *cpMock, err error) { cp.changesErr = err }, errors.New("changesErr")},
		{func(cp *cpMock, err error) { cp.waitChSetErr = err }, errors.New("waitChSetErr")},
	}
	for _, tc := range cases {
		cp := &cpMock{}
		tc.errProv(cp, tc.err)

		_, err := New(cp, StackTemplate{})

		if tc.err != err {
			t.Errorf("Expected to get error %v. Got %v", tc.err, err)
		}

	}
}

func TestChangeSetExecutionErrors(t *testing.T) {
	cases := []struct {
		errProv func(*cpMock, error)
		err     error
	}{
		{func(cp *cpMock, err error) { cp.execErr = err }, errors.New("execErr")},
		{func(cp *cpMock, err error) { cp.waitStackErr = err }, errors.New("waitStackErr")},
	}
	for _, tc := range cases {
		cp := &cpMock{}
		tc.errProv(cp, tc.err)

		cs, _ := New(cp, StackTemplate{})
		err := cs.Exec()

		if tc.err != err {
			t.Errorf("Expected to get error %v. Got %v", tc.err, err)
		}

	}
}

type cpMock struct {
	sync.Mutex
	waitStackFunc   func() error
	events          []StackEvent
	chSetID         string
	requiredParams  []string
	submittedParams map[string]string
	outputs         []StackOutput
	validationErr   error
	createErr       error
	changesErr      error
	waitChSetErr    error
	execErr         error
	waitStackErr    error
	executed        bool
	name            string
	body            string
	blocked         []string
}

func (cpm *cpMock) ValidateTemplate(tplBody string) ([]string, error) {
	return cpm.requiredParams, cpm.validationErr
}
func (cpm *cpMock) CreateChangeSet(stackName string, tplBody string, params map[string]string) (string, error) {
	cpm.name = stackName
	cpm.body = tplBody
	cpm.submittedParams = params

	if cpm.chSetID == "" {
		return "ID", cpm.createErr
	}
	return cpm.chSetID, cpm.createErr
}
func (cpm *cpMock) WaitChangeSetCreated(ID string) error {
	return cpm.waitChSetErr
}
func (cpm *cpMock) ChangeSetChanges(ID string) ([]Change, error) {
	return []Change{}, cpm.changesErr
}
func (cpm *cpMock) ExecuteChangeSet(ID string) error {
	cpm.executed = true
	return cpm.execErr
}
func (cpm *cpMock) WaitStack(stackName string) error {
	if cpm.waitStackFunc != nil {
		return cpm.waitStackFunc()
	}
	return cpm.waitStackErr
}
func (cpm *cpMock) StackEvents(stackName string) ([]StackEvent, error) {
	cpm.Lock()
	defer cpm.Unlock()
	return cpm.events, nil
}
func (cpm *cpMock) StackOutputs(stackName string) ([]StackOutput, error) {
	return cpm.outputs, nil
}
func (cpm *cpMock) StackResources(stackName string) ([]StackResource, error) {
	return nil, nil
}
func (cpm *cpMock) BlockResource(stackName string, resource string) error {
	if nil == cpm.blocked {
		cpm.blocked = make([]string, 0)
	}
	cpm.blocked = append(cpm.blocked, resource)
	return nil
}
func (cpm *cpMock) UnblockResource(stackName string, resource string) error {
	return nil
}
func (cpm *cpMock) AssertBlocked(t *testing.T, resources []string) {
	sort.Strings(cpm.blocked)
	sort.Strings(resources)

	if !reflect.DeepEqual(cpm.blocked, resources) {
		t.Errorf("Resources %v expected to be blocked. Got %v", resources, cpm.blocked)
	}
}
