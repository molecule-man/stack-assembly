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

type stackData struct {
	Outputs map[string]map[string]string
	Params  map[string]string
}

// SyncAll syncs all the provided templates one by one
func (s *Service) SyncAll(tpls map[string]StackTemplate, globalParams map[string]string) error {
	ordered, err := s.order(tpls)

	if err != nil {
		return err
	}

	data := stackData{}
	data.Outputs = make(map[string]map[string]string)

	for _, id := range ordered {
		t := tpls[id]

		data.Params = globalParams

		if err := s.initParams(&t, data); err != nil {
			return err
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

		data.Outputs[id] = make(map[string]string, len(out))

		for _, v := range out {
			data.Outputs[id][v.Key] = v.Value
		}
	}
	return nil
}

// Sync syncs
func (s *Service) Sync(tpl StackTemplate) error {
	data := struct{ Params map[string]string }{tpl.Params}

	if err := applyTemplating(&tpl.Name, tpl.Name, data); err != nil {
		return err
	}

	if err := applyTemplating(&tpl.Body, tpl.Body, data); err != nil {
		return err
	}

	if err := s.execSync(tpl); err != nil {
		return err
	}

	return s.block(tpl)
}

type StackInfo struct {
	Name      string
	Resources []StackResource
	Outputs   []StackOutput
	Events    []StackEvent
}

func (s *Service) Info(tpl StackTemplate, globalParams map[string]string) (StackInfo, error) {
	si := StackInfo{}

	data := stackData{Params: globalParams}

	if err := s.initParams(&tpl, data); err != nil {
		return si, err
	}

	if err := applyTemplating(&tpl.Name, tpl.Name, data); err != nil {
		return si, err
	}

	si.Name = tpl.Name

	var err error

	if si.Resources, err = s.CloudProvider.StackResources(tpl.Name); err != nil {
		return si, err
	}

	if si.Outputs, err = s.CloudProvider.StackOutputs(tpl.Name); err != nil {
		return si, err
	}

	si.Events, err = s.CloudProvider.StackEvents(tpl.Name)

	return si, err
}

func (s Service) execSync(tpl StackTemplate) error {
	log := s.logFunc(tpl.Name)

	log("Syncing template")

	chSet, err := New(s.CloudProvider, tpl,
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
		return errors.New("Sync is cancelled")
	}

	err = chSet.Exec()

	log("Sync is finished")

	return err
}

func (s Service) block(tpl StackTemplate) error {
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

func (s Service) order(tpls map[string]StackTemplate) ([]string, error) {
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
				return []string{}, err
			}

			dg.Add(id, deps)
		}
	}

	return dg.Resolve()
}

func (s Service) initParams(tpl *StackTemplate, data stackData) error {
	if tpl.Params == nil {
		tpl.Params = make(map[string]string)
	}
	for k, v := range data.Params {
		if _, ok := tpl.Params[k]; !ok {
			tpl.Params[k] = v
		}
	}
	for k, v := range tpl.Params {
		var parsed string
		if err := applyTemplating(&parsed, v, data); err != nil {
			return err
		}
		tpl.Params[k] = parsed
	}
	return nil
}

func (s Service) logFunc(logID string) func(string) {
	return func(msg string) {
		s.Log.Print(fmt.Sprintf("[%s] %s", logID, msg))
	}
}
