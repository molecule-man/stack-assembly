package claws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/molecule-man/claws/cloudprov"
)

// ChangeSet represents aws changeset
type ChangeSet struct {
	Changes   []cloudprov.Change
	StackName string
	ID        string
	cf        *cloudformation.CloudFormation
	cp        cloudprov.CloudProvider
}

// New creates a new ChangeSet
func New(cp cloudprov.CloudProvider, stackName string, tplBody string, userParams map[string]string) (*ChangeSet, error) {
	sess := session.Must(session.NewSession())
	cf := cloudformation.New(sess)

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
		cf:        cf,
		cp:        cp,
	}

	err = chSet.initialize(tplBody, params)
	return chSet, err
}

// Exec executes the ChangeSet
func (cs *ChangeSet) Exec() error {
	_, err := cs.cf.ExecuteChangeSet(&cloudformation.ExecuteChangeSetInput{
		ChangeSetName: &cs.ID,
	})

	if err != nil {
		return err
	}

	return cs.cp.WaitStack(cs.StackName)
}

// EventsTracker creates a new event tracker for the executed stack
func (cs *ChangeSet) EventsTracker() EventsTracker {
	return EventsTracker{
		cp:        cs.cp,
		stackName: cs.StackName,
	}
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
