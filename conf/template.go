package conf

import (
	"bytes"
	"os/exec"
	"strings"
	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/sts"
)

type tplData struct {
	AWS struct {
		AccountID string
		Region    string
	}
	Params map[string]string
}

func applyTemplating(cfg *Config) error {
	data, err := newTplData(cfg)
	if err != nil {
		return err
	}

	for i, stackCfg := range cfg.Stacks {
		if stackCfg.Parameters == nil {
			stackCfg.Parameters = make(map[string]string, len(cfg.Parameters))
		}

		for k, v := range cfg.Parameters {
			if _, ok := stackCfg.Parameters[k]; !ok {
				stackCfg.Parameters[k] = v
			}
		}

		data.Params = stackCfg.Parameters

		if err := templatizeMap(&stackCfg.Parameters, data); err != nil {
			return err
		}
		data.Params = stackCfg.Parameters

		if err := templatizeMap(&stackCfg.Tags, data); err != nil {
			return err
		}

		err := parseTpl(&stackCfg.Name, stackCfg.Name, data)
		if err != nil {
			return err
		}

		err = templatizeRollbackConfig(stackCfg.RollbackConfiguration, data)
		if err != nil {
			return err
		}

		cfg.Stacks[i] = stackCfg
	}
	return nil
}

func newTplData(cfg *Config) (tplData, error) {
	data := tplData{}

	opts := awsOpts(*cfg)
	sess := session.Must(session.NewSessionWithOptions(opts))
	callerIdent, err := sts.New(sess).GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return data, err
	}

	data.AWS.Region = aws.StringValue(sess.Config.Region)
	data.AWS.AccountID = aws.StringValue(callerIdent.Account)

	return data, nil
}

func templatizeRollbackConfig(rlbCfg *cloudformation.RollbackConfiguration, data tplData) error {
	if rlbCfg == nil {
		return nil
	}

	for _, t := range rlbCfg.RollbackTriggers {
		err := parseTpl(t.Arn, aws.StringValue(t.Arn), data)
		if err != nil {
			return err
		}
	}

	return nil
}

func templatizeMap(m *map[string]string, data tplData) error {
	if *m == nil {
		*m = map[string]string{}
	}

	for k, v := range *m {
		var parsed string
		if err := parseTpl(&parsed, v, data); err != nil {
			return err
		}
		(*m)[k] = parsed
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
