package stackassembly

import (
	"bytes"
	"io/ioutil"
	"text/template"

	"github.com/molecule-man/stack-assembly/depgraph"
)

type StackConfig struct {
	Name       string
	Path       string
	Body       string
	Parameters map[string]string
	Tags       map[string]string
	DependsOn  []string
	Blocked    []string
}

type Stack struct {
	ID         string
	Name       string
	Parameters map[string]string
	Tags       map[string]string
	DependsOn  []string
	Blocked    []string

	body string
	path string
}

func NewStack(id string, stackCfg StackConfig, globalParameters map[string]string) (Stack, error) {
	stack := Stack{}

	stack.path = stackCfg.Path

	stack.ID = id
	stack.DependsOn = stackCfg.DependsOn
	stack.Blocked = stackCfg.Blocked
	stack.Tags = stackCfg.Tags

	stack.Parameters = make(map[string]string, len(globalParameters)+len(stackCfg.Parameters))

	for k, v := range globalParameters {
		if _, ok := stackCfg.Parameters[k]; !ok {
			stack.Parameters[k] = v
		}
	}

	data := struct{ Params map[string]string }{}
	data.Params = stack.Parameters

	for k, v := range stackCfg.Parameters {
		var parsed string
		if err := applyTemplating(&parsed, v, data); err != nil {
			return stack, err
		}
		stack.Parameters[k] = parsed
	}

	data.Params = stack.Parameters

	stack.Tags = make(map[string]string, len(stackCfg.Tags))
	for k, v := range stackCfg.Tags {
		var parsed string
		if err := applyTemplating(&parsed, v, data); err != nil {
			return stack, err
		}
		stack.Tags[k] = parsed
	}

	err := applyTemplating(&stack.Name, stackCfg.Name, data)

	return stack, err
}

func (t *Stack) Body() (string, error) {
	if t.body != "" {
		return t.body, nil
	}
	tplBody, err := ioutil.ReadFile(t.path)
	if err != nil {
		return "", err
	}

	data := struct{ Params map[string]string }{}
	data.Params = t.Parameters
	err = applyTemplating(&t.body, string(tplBody), data)
	return t.body, err
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

func SortStacksByExecOrder(stacks []Stack) error {
	dg := depgraph.DepGraph{}

	stacksMap := make(map[string]Stack, len(stacks))

	for _, stack := range stacks {
		dg.Add(stack.ID, stack.DependsOn)
		stacksMap[stack.ID] = stack
	}

	orderedIds, err := dg.Resolve()
	if err != nil {
		return err
	}

	for i, id := range orderedIds {
		stacks[i] = stacksMap[id]
	}

	return nil
}
