package claws

import (
	"time"
)

type stackEventListener func(StackEvent)

// ChangeSet represents aws changeset
type ChangeSet struct {
	Changes   []Change
	StackName string
	ID        string
	cp        CloudProvider
	listeners []stackEventListener
	sleep     time.Duration
}

// StackTemplate encapsulates information about stack template
type StackTemplate struct {
	Name      string
	Body      string
	Params    map[string]string
	DependsOn []string
}

// Option is function that can be used to configure new change set
type Option func(cs *ChangeSet)

// New creates a new ChangeSet
func New(cp CloudProvider, tpl StackTemplate, opts ...Option) (*ChangeSet, error) {
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
		StackName: tpl.Name,
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
func WithEventSubscriber(cb stackEventListener) Option {
	return func(cs *ChangeSet) {
		cs.listeners = append(cs.listeners, cb)
	}
}

// WithEventSleep is an option that configures sleep time to use when polling
// for events
func WithEventSleep(t time.Duration) Option {
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

	operation := CreateOperation
	if exists {
		operation = UpdateOperation
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
