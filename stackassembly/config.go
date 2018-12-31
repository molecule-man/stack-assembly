package stackassembly

import (
	"bytes"
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

// Config is a struct holding stacks configurations
type Config struct {
	Parameters map[string]string
	Stacks     map[string]StackConfig
}

type Stack struct {
	Name       string
	Parameters map[string]string
	Tags       map[string]string
	Blocked    []string

	rawBody string
}

func NewStack(stackCfg StackConfig, globalParameters map[string]string) (Stack, error) {
	stack := Stack{}

	// TODO this doesn't belong here
	stack.rawBody = stackCfg.Body

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

func StacksSortedByExecOrder(cfg Config) ([]Stack, error) {
	dg := depgraph.DepGraph{}

	stacksMap := make(map[string]Stack, len(cfg.Stacks))
	stacks := make([]Stack, 0, len(cfg.Stacks))

	for id, stackCfg := range cfg.Stacks {
		dg.Add(id, stackCfg.DependsOn)
		stack, err := NewStack(stackCfg, cfg.Parameters)
		if err != nil {
			return stacks, err
		}

		stacksMap[id] = stack
	}

	orderedIds, err := dg.Resolve()
	if err != nil {
		return stacks, err
	}

	for _, id := range orderedIds {
		stacks = append(stacks, stacksMap[id])
	}

	return stacks, nil
}
