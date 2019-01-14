package conf

import (
	"bytes"
	"os/exec"
	"strings"
	"text/template"
)

func applyTemplating(cfg *Config) error {
	for i, stackCfg := range cfg.Stacks {
		parameters := make(map[string]string, len(cfg.Parameters)+len(stackCfg.Parameters))

		for k, v := range cfg.Parameters {
			if _, ok := stackCfg.Parameters[k]; !ok {
				parameters[k] = v
			}
		}

		data := struct{ Params map[string]string }{}
		data.Params = parameters

		for k, v := range stackCfg.Parameters {
			var parsed string
			if err := parseTpl(&parsed, v, data); err != nil {
				return err
			}
			parameters[k] = parsed
		}

		stackCfg.Parameters = parameters
		data.Params = parameters

		tags := make(map[string]string, len(stackCfg.Tags))
		for k, v := range stackCfg.Tags {
			var parsed string
			if err := parseTpl(&parsed, v, data); err != nil {
				return err
			}
			tags[k] = parsed
		}
		stackCfg.Tags = tags

		err := parseTpl(&stackCfg.Name, stackCfg.Name, data)
		if err != nil {
			return err
		}

		cfg.Stacks[i] = stackCfg
	}
	return nil
}

func parseTpl(parsed *string, tpl string, data interface{}) error {
	t, err := template.New(tpl).Funcs(template.FuncMap{
		"Exec": func(cmd string, args ...string) (string, error) {
			out, err := exec.Command(cmd, args...).CombinedOutput()
			if err != nil {
				return "", err
			}

			return strings.TrimSpace(string(out)), nil
		},
	}).Parse(tpl)
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
