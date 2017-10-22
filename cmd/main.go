// Package main provides cmd claws application
package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/molecule-man/claws/awsprov"
	"github.com/molecule-man/claws/claws"
	"github.com/molecule-man/claws/cli"
)

type (
	// Config is a struct holding templates configurations
	Config struct {
		Parameters map[string]string
		Templates  map[string]TemplateConfig
	}

	// TemplateConfig is a configuration of a stack template
	TemplateConfig struct {
		Path       string
		Name       string
		Parameters map[string]string
		DependsOn  []string
		Blocked    []string
	}
)

// merge merges otherConfig into this config
func (c *Config) merge(otherConfig Config) {
	if c.Parameters == nil {
		c.Parameters = make(map[string]string)
	}
	for k, v := range otherConfig.Parameters {
		c.Parameters[k] = v
	}

	if c.Templates == nil {
		c.Templates = make(map[string]TemplateConfig)
	}
	for k, tc := range otherConfig.Templates {
		t := c.Templates[k]
		t.merge(tc)
	}
}

func (tc *TemplateConfig) merge(otherTpl TemplateConfig) {
	if otherTpl.Path != "" {
		tc.Path = otherTpl.Path
	}

	if otherTpl.Name != "" {
		tc.Name = otherTpl.Name
	}

	for k, v := range otherTpl.Parameters {
		tc.Parameters[k] = v
	}
}

func main() {
	tpl := flag.String("tpl", "", "CF tpl")
	cfgFile := flag.String("cfg", "", "CF tpl")
	stackName := flag.String("stack", "", "Stack name")

	flag.Parse()

	cfg := readConfigs(cfgFile)

	if tpl != nil && *tpl != "" {
		id := *stackName
		cfg.Templates = map[string]TemplateConfig{
			id: {
				Path: *tpl,
				Name: *stackName,
			},
		}
	}

	serv := claws.Service{
		Approver:      &cli.Approval{},
		Log:           log.New(os.Stderr, "", log.LstdFlags),
		CloudProvider: awsprov.New(),
	}

	tpls := make(map[string]claws.StackTemplate)
	for i, template := range cfg.Templates {
		tplBody, err := ioutil.ReadFile(template.Path)

		if err != nil {
			log.Fatal(err)
		}

		tpls[i] = claws.StackTemplate{
			Name:      template.Name,
			Body:      string(tplBody),
			Params:    template.Parameters,
			DependsOn: template.DependsOn,
			Blocked:   template.Blocked,
		}
	}
	if err := serv.SyncAll(tpls, cfg.Parameters); err != nil {
		log.Fatal(err)
	}
}

func readConfigs(cfgFiles *string) Config {
	mainConfig := Config{}

	if _, err := os.Stat("Claws.toml"); err == nil {
		mainConfig = readConfig("Claws.toml")
	}

	if cfgFiles == nil || *cfgFiles == "" {
		return mainConfig
	}

	for _, cf := range strings.Split(*cfgFiles, " ") {
		mainConfig.merge(readConfig(cf))
	}

	return mainConfig
}

func readConfig(f string) Config {
	cfg := Config{}
	_, err := toml.DecodeFile(f, &cfg)

	if err != nil {
		log.Fatal(err)
	}

	return cfg
}
