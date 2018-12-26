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

	for _, th := range ordered {
		if err = s.execSync(th); err != nil {
			return err
		}

		err = s.block(th)
		if err != nil {
			return err
		}
	}
	return nil
}

type StackInfo struct {
	Name      string
	Resources []StackResource
	Outputs   []StackOutput
	Events    []StackEvent
}

func (s *Service) Info(tpl StackTemplate, globalParams map[string]string) (StackInfo, error) {
	si := StackInfo{}

	th, err := NewThing(tpl, globalParams)

	if err != nil {
		return si, err
	}

	si.Name = th.Name

	if si.Resources, err = s.CloudProvider.StackResources(th.Name); err != nil {
		return si, err
	}

	if si.Outputs, err = s.CloudProvider.StackOutputs(th.Name); err != nil {
		return si, err
	}

	si.Events, err = s.CloudProvider.StackEvents(th.Name)

	return si, err
}

func (s Service) execSync(th TheThing) error {
	log := s.logFunc(th.Name)

	log("Syncing template")

	chSet, err := New(s.CloudProvider, th,
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

func (s Service) block(tpl TheThing) error {
	log := s.logFunc(tpl.Name)

	for _, r := range tpl.Blocked {
		log(fmt.Sprintf("Blocking resource %s", r))
		err := s.CloudProvider.BlockResource(tpl.Name, r)

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
