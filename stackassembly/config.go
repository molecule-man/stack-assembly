package stackassembly

import (
	"bytes"
	"text/template"

	"github.com/molecule-man/stack-assembly/depgraph"
)

// StackTemplate encapsulates information about stack template
type StackTemplate struct {
	Name       string
	Path       string
	Body       string
	Parameters map[string]string
	DependsOn  []string
	Blocked    []string
}

// Config is a struct holding templates configurations
type Config struct {
	Parameters map[string]string
	Templates  map[string]StackTemplate
}

type TheThing struct {
	Name       string
	Parameters map[string]string
	Blocked    []string

	rawBody string
}

func NewThing(tpl StackTemplate, globalParameters map[string]string) (TheThing, error) {
	t := TheThing{}

	// TODO this doesn't belong here
	t.rawBody = tpl.Body

	t.Blocked = tpl.Blocked

	t.Parameters = make(map[string]string, len(globalParameters)+len(tpl.Parameters))

	for k, v := range globalParameters {
		if _, ok := tpl.Parameters[k]; !ok {
			t.Parameters[k] = v
		}
	}

	data := struct{ Params map[string]string }{}
	data.Params = t.Parameters

	for k, v := range tpl.Parameters {
		var parsed string
		if err := applyTemplating(&parsed, v, data); err != nil {
			return t, err
		}
		t.Parameters[k] = parsed
	}

	data.Params = t.Parameters

	err := applyTemplating(&t.Name, tpl.Name, data)

	return t, err
}

func (t *TheThing) Body() (string, error) {
	var body string
	data := struct{ Params map[string]string }{}
	data.Params = t.Parameters
	err := applyTemplating(&body, t.rawBody, data)
	return body, err
}

func applyTemplating(parsed *string, tpl string, data interface{}) error {
	t, err := template.New(tpl).Parse(tpl)
	if err != nil {
		return err
	}
	var buff bytes.Buffer

	if err := t.Execute(&buff, data); err != nil {
		return err
	}

	*parsed = buff.String()
	return nil
}

func sortedByExecOrder(cfg Config) ([]TheThing, error) {
	dg := depgraph.DepGraph{}

	tplsMap := make(map[string]TheThing, len(cfg.Templates))
	tpls := make([]TheThing, 0, len(cfg.Templates))

	for id, tpl := range cfg.Templates {
		dg.Add(id, tpl.DependsOn)
		t, err := NewThing(tpl, cfg.Parameters)
		if err != nil {
			return tpls, err
		}

		tplsMap[id] = t
	}

	orderedIds, err := dg.Resolve()
	if err != nil {
		return tpls, err
	}

	for _, id := range orderedIds {
		tpls = append(tpls, tplsMap[id])
	}

	return tpls, nil
}
