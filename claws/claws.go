package claws

import (
	"time"

	"github.com/molecule-man/claws/cloudprov"
)

type stackEventListener func(cloudprov.StackEvent)

// ChangeSet represents aws changeset
type ChangeSet struct {
	Changes   []cloudprov.Change
	StackName string
	ID        string
	cp        cloudprov.CloudProvider
	listeners []stackEventListener
	sleep     time.Duration
}

// StackTemplate encapsulates information about stack template
type StackTemplate struct {
	StackName string
	Body      string
	Params    map[string]string
}

type option func(cs *ChangeSet)

// New creates a new ChangeSet
func New(cp cloudprov.CloudProvider, tpl StackTemplate, opts ...option) (*ChangeSet, error) {
	requiredParams, err := cp.ValidateTemplate(tpl.Body)

	if err != nil {
		return nil, err
	}

	params := make(map[string]string, len(requiredParams))

	for _, p := range requiredParams {
		if v, ok := tpl.Params[p]; ok {
			params[p] = v
		}
	}

	chSet := &ChangeSet{
		StackName: tpl.StackName,
		cp:        cp,
		sleep:     time.Second,
	}

	for _, opt := range opts {
		opt(chSet)
	}

	err = chSet.initialize(tpl.Body, params)
	return chSet, err
}

// WithEventSubscriber is an option that configures chage set to add stack
// events listener
func WithEventSubscriber(cb stackEventListener) option {
	return func(cs *ChangeSet) {
		cs.listeners = append(cs.listeners, cb)
	}
}

func WithEventSleep(t time.Duration) option {
	return func(cs *ChangeSet) {
		cs.sleep = t
	}
}

// Exec executes the ChangeSet
func (cs *ChangeSet) Exec() error {
	et := EventsTracker{
		cp:        cs.cp,
		stackName: cs.StackName,
	}

	events := et.StartTracking()

	err := cs.cp.ExecuteChangeSet(cs.ID)

	if err != nil {
		et.StopTracking()
		return err
	}

	errCh := make(chan error)

	go func() {
		err := cs.cp.WaitStack(cs.StackName)
		et.StopTracking()
		errCh <- err
	}()

	for event := range events {
		for _, cb := range cs.listeners {
			cb(event)
		}
	}

	return <-errCh
}

func (cs *ChangeSet) initialize(tplBody string, params map[string]string) error {
	exists, err := cs.cp.StackExists(cs.StackName)

	if err != nil {
		return err
	}

	operation := cloudprov.CreateOperation
	if exists {
		operation = cloudprov.UpdateOperation
	}

	cs.ID, err = cs.cp.CreateChangeSet(cs.StackName, tplBody, params, operation)

	if err != nil {
		return err
	}

	if _, err = cs.cp.ChangeSetChanges(cs.ID); err != nil {
		return err
	}

	if err = cs.cp.WaitChangeSetCreated(cs.ID); err != nil {
		return err
	}

	cs.Changes, err = cs.cp.ChangeSetChanges(cs.ID)

	return err
}
