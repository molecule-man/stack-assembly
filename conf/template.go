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

	*cfg, err = templatizeStackConfig(*cfg, data)
	return err
}

func templatizeStackConfig(cfg Config, data tplData) (Config, error) {
	if err := templatizeParams(&cfg.Parameters, data); err != nil {
		return cfg, err
	}

	data.Params = cfg.Parameters

	if err := templatizeMap(&cfg.Tags, data); err != nil {
		return cfg, err
	}

	if err := parseTpl(&cfg.Name, cfg.Name, data); err != nil {
		return cfg, err
	}

	if err := parseTpl(&cfg.Body, cfg.Body, data); err != nil {
		return cfg, err
	}

	if err := templatizeRollbackConfig(cfg.RollbackConfiguration, data); err != nil {
		return cfg, err
	}

	for i, nestedCfg := range cfg.Stacks {
		templatizedCfg, err := templatizeStackConfig(nestedCfg, data)
		if err != nil {
			return cfg, err
		}

		cfg.Stacks[i] = templatizedCfg
	}

	return cfg, nil
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
	data.Params = map[string]string{}

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

func templatizeParams(parameters *map[string]string, data tplData) error {
	if *parameters == nil {
		*parameters = make(map[string]string, len(data.Params))
	}

	for k, v := range data.Params {
		if _, ok := (*parameters)[k]; !ok {
			(*parameters)[k] = v
		}
	}

	return templatizeMap(parameters, data)
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
