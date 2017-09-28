// Package claws provides claws stuff
package claws

import (
	"bytes"
	"errors"
	"fmt"
	"text/template"
	"text/template/parse"

	"github.com/molecule-man/claws/depgraph"
)

const stackOutputVarName = "Outputs"

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
	dg := depgraph.DepGraph{}

	for id, t := range tpls {
		dg.Add(id, t.DependsOn)

		templatableFields := make([]string, 0, len(t.Params)+2)
		for _, v := range t.Params {
			templatableFields = append(templatableFields, v)
		}
		templatableFields = append(templatableFields, t.Name, t.Body)

		for _, f := range templatableFields {
			deps, err := dependencies(f)

			if err != nil {
				return err
			}

			dg.Add(id, deps)
		}
	}

	ordered, err := dg.Resolve()

	if err != nil {
		return err
	}

	data := struct {
		Outputs map[string]map[string]string
		Params  map[string]string
	}{}
	data.Outputs = make(map[string]map[string]string)

	for _, id := range ordered {
		t := tpls[id]
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
			data.Params = globalParams
			if err := applyTemplating(&parsed, v, data); err != nil {
				return err
			}
			t.Params[k] = parsed
		}

		data.Params = t.Params

		if err := applyTemplating(&t.Name, t.Name, data); err != nil {
			return err
		}

		err := s.Sync(t)

		if err != nil {
			return err
		}

		out, err := s.CloudProvider.StackOutputs(t.Name)

		if err != nil {
			return err
		}

		data.Outputs[id] = out
	}
	return nil
}

// Sync syncs
func (s *Service) Sync(tpl StackTemplate) error {
	log := s.logFunc(tpl.Name)

	data := struct{ Params map[string]string }{tpl.Params}

	if err := applyTemplating(&tpl.Name, tpl.Name, data); err != nil {
		return err
	}
	if err := applyTemplating(&tpl.Body, tpl.Body, data); err != nil {
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

func applyTemplating(parsed *string, tpl string, params interface{}) error {
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

func dependencies(tpl string) ([]string, error) {
	deps := []string{}

	t, err := template.New(tpl).Parse(tpl)
	if err != nil {
		return deps, err
	}

	for _, n := range t.Tree.Root.Nodes {
		node, ok := n.(*parse.ActionNode)
		if !ok {
			continue
		}

		for _, c := range node.Pipe.Cmds {
			fn, ok := c.Args[0].(*parse.FieldNode)
			if !ok || len(fn.Ident) < 1 || fn.Ident[0] != stackOutputVarName {
				continue
			}

			deps = append(deps, fn.Ident[1])
		}
	}

	return deps, nil
}

func (s *Service) logFunc(logID string) func(string) {
	return func(msg string) {
		s.Log.Print(fmt.Sprintf("[%s] %s", logID, msg))
	}
}
