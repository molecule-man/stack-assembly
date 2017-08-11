// Package claws provides claws stuff
package claws

import (
	"bytes"
	"errors"
	"fmt"
	"text/template"
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

// SyncAll syncs all the provided templates one by one
func (s *Service) SyncAll(tpls map[string]StackTemplate, globalParams map[string]string) error {
	for _, t := range tpls {
		if t.Params == nil {
			t.Params = make(map[string]string)
		}
		for k, v := range globalParams {
			if _, ok := t.Params[k]; !ok {
				t.Params[k] = v
			}
		}
		for k, v := range t.Params {
			var parsed string
			if err := applyTemplating(&parsed, v, globalParams); err != nil {
				return err
			}
			t.Params[k] = parsed
		}

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

	if err := applyTemplating(&tpl.Name, tpl.Name, tpl.Params); err != nil {
		return err
	}
	if err := applyTemplating(&tpl.Body, tpl.Body, tpl.Params); err != nil {
		return err
	}

	log("Syncing template")

	chSet, err := New(s.CloudProvider, tpl,
		WithEventSubscriber(func(e StackEvent) {
			log(fmt.Sprintf("[%s] [%s] [%s] %s",
				e.ResourceType, e.Status, e.LogicalResourceID, e.StatusReason,
			))
		}),
	)

	if err != nil {
		if err == ErrNoChange {
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

func applyTemplating(parsed *string, tpl string, params map[string]string) error {
	t, err := template.New(tpl).Parse(tpl)
	if err != nil {
		return err
	}
	var buff bytes.Buffer

	if err := t.Execute(&buff, params); err != nil {
		return err
	}

	*parsed = buff.String()
	return nil
}

func (s *Service) logFunc(logID string) func(string) {
	return func(msg string) {
		s.Log.Print(fmt.Sprintf("[%s] %s", logID, msg))
	}
}
