// Package claws provides claws stuff
package claws

import (
	"errors"
	"fmt"

	"github.com/molecule-man/claws/cloudprov"
)

type approver interface {
	Approve() bool
}
type logger interface {
	Print(v ...interface{})
}
type changePresenter interface {
	ShowChanges([]cloudprov.Change)
}

// Service orchestrates synchronization of templates
type Service struct {
	Approver        approver
	Log             logger
	ChangePresenter changePresenter
	CloudProvider   cloudprov.CloudProvider
}

// Sync syncs
func (s *Service) Sync(stackName, body string, params map[string]string) error {
	log := s.logFunc(stackName)

	log("Syncing template")

	chSet, err := New(
		s.CloudProvider,
		StackTemplate{
			StackName: stackName,
			Body:      body,
			Params:    params,
		},
		WithEventSubscriber(func(e cloudprov.StackEvent) {
			log(fmt.Sprintf("[%s] [%s] [%s] %s",
				e.ResourceType, e.Status, e.LogicalResourceID, e.StatusReason,
			))
		}),
	)

	if err != nil {
		if err == cloudprov.ErrNoChange {
			log("No changes to be synced")
			return nil
		}
		return err
	}

	log("Change set is created")

	s.ChangePresenter.ShowChanges(chSet.Changes)

	if !s.Approver.Approve() {
		return errors.New("Sync is cancelled")
	}

	err = chSet.Exec()

	log("Sync is finished")

	return err
}

func (s *Service) logFunc(logID string) func(string) {
	return func(msg string) {
		s.Log.Print(fmt.Sprintf("[%s] %s", logID, msg))
	}
}
