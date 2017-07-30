package claws

import (
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

type CloudProviderMock struct {
	sync.Mutex
	waitStackFunc func() error
	events        []cloudprov.StackEvent
}

func (cpm *CloudProviderMock) ValidateTemplate(tplBody string) ([]string, error) {
	return []string{}, nil
}
func (cpm *CloudProviderMock) StackExists(stackName string) (bool, error) {
	return false, nil
}
func (cpm *CloudProviderMock) CreateChangeSet(stackName string, tplBody string, params map[string]string, op cloudprov.ChangeSetOperation) (string, error) {
	return "ID", nil
}
func (cpm *CloudProviderMock) WaitChangeSetCreated(ID string) error {
	return nil
}
func (cpm *CloudProviderMock) ChangeSetChanges(ID string) ([]cloudprov.Change, error) {
	return []cloudprov.Change{}, nil
}
func (cpm *CloudProviderMock) ExecuteChangeSet(ID string) error {
	return nil
}
func (cpm *CloudProviderMock) WaitStack(stackName string) error {
	if cpm.waitStackFunc != nil {
		return cpm.waitStackFunc()
	}
	return nil
}
func (cpm *CloudProviderMock) StackEvents(stackName string) ([]cloudprov.StackEvent, error) {
	cpm.Lock()
	defer cpm.Unlock()
	return cpm.events, nil
}
