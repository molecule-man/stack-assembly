// Package stackassembly provides stack-assembly core functionality
package stackassembly

import (
	"errors"
	"fmt"
)

type approver interface {
	Approve([]Change) bool
}

type logger interface {
	Print(v ...interface{})
}

// Service orchestrates synchronization of templates
type Service struct {
	Approver      approver
	Log           logger
	CloudProvider CloudProvider
}

// Sync syncs all the provided templates one by one
func (s *Service) Sync(cfg Config) error {
	ordered, err := sortedByExecOrder(cfg)

	if err != nil {
		return err
	}

	for _, stack := range ordered {
		if err = s.execSync(stack); err != nil {
			return err
		}

		err = s.block(stack)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s Service) execSync(stack Stack) error {
	log := s.logFunc(stack.Name)

	log("Syncing template")

	chSet, err := New(s.CloudProvider, stack,
		WithEventSubscriber(func(e StackEvent) {
			log(fmt.Sprintf("[%s] [%s] [%s] %s",
				e.ResourceType, e.Status, e.LogicalResourceID, e.StatusReason,
			))
		}),
	)

	if err == ErrNoChange {
		log("No changes to be synced")
		return nil
	}

	if err != nil {
		return err
	}

	log(fmt.Sprintf("Change set is created: %s", chSet.ID))

	if !s.Approver.Approve(chSet.Changes) {
		return errors.New("sync is cancelled")
	}

	err = chSet.Exec()

	log("Sync is finished")

	return err
}

func (s Service) block(stack Stack) error {
	log := s.logFunc(stack.Name)

	for _, r := range stack.Blocked {
		log(fmt.Sprintf("Blocking resource %s", r))
		err := s.CloudProvider.BlockResource(stack.Name, r)

		if err != nil {
			return err
		}
	}

	return nil
}

func (s Service) logFunc(logID string) func(string) {
	return func(msg string) {
		s.Log.Print(fmt.Sprintf("[%s] %s", logID, msg))
	}
}
