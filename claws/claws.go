package claws

import (
	"github.com/molecule-man/claws/cloudprov"
)

type stackEventListener func(string)

// ChangeSet represents aws changeset
type ChangeSet struct {
	Changes   []cloudprov.Change
	StackName string
	ID        string
	cp        cloudprov.CloudProvider
	listeners []stackEventListener
}

// New creates a new ChangeSet
func New(cp cloudprov.CloudProvider, stackName string, tplBody string, userParams map[string]string) (*ChangeSet, error) {
	tplParams, err := cp.ValidateTemplate(tplBody)

	if err != nil {
		return nil, err
	}

	params := make(map[string]string, len(tplParams))

	for _, p := range tplParams {
		if v, ok := userParams[p]; ok {
			params[p] = v
		}
	}

	chSet := &ChangeSet{
		StackName: stackName,
		cp:        cp,
	}

	err = chSet.initialize(tplBody, params)
	return chSet, err
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

// Subscribe stack events listener
func (cs *ChangeSet) Subscribe(cb stackEventListener) {
	cs.listeners = append(cs.listeners, cb)
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
