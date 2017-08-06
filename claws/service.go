// Package claws provides claws stuff
package claws

import (
	"errors"
	"fmt"

	"github.com/molecule-man/claws/cloudprov"
)

type approver interface {
	Approve([]cloudprov.Change) bool
}
type logger interface {
	Print(v ...interface{})
}

// Service orchestrates synchronization of templates
type Service struct {
	Approver      approver
	Log           logger
	CloudProvider cloudprov.CloudProvider
}

// SyncAll syncs all the provided templates one by one
func (s *Service) SyncAll(tpls map[string]StackTemplate) error {
	for _, t := range tpls {
		err := s.Sync(t)

		if err != nil {
			return err
		}
	}
	return nil
}

// Sync syncs
func (s *Service) Sync(tpl StackTemplate) error {
	log := s.logFunc(tpl.Name)

	log("Syncing template")

	chSet, err := New(s.CloudProvider, tpl,
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

	if !s.Approver.Approve(chSet.Changes) {
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
