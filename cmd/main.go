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
	cfgFilesFlags []string
)

func (i *cfgFilesFlags) String() string {
	return strings.Join(*i, ",")
}

func (i *cfgFilesFlags) Set(val string) error {
	*i = append(*i, val)
	return nil
}

func main() {
	var cfgFiles cfgFilesFlags
	tpl := flag.String("tpl", "", "CF tpl")
	flag.Var(&cfgFiles, "f", "CF configs")
	stackName := flag.String("stack", "", "Stack name")

	flag.Parse()

	cfg := readConfigs(cfgFiles)

	if tpl != nil && *tpl != "" {
		id := *stackName
		cfg.Templates = map[string]TemplateConfig{
			id: {
				Path: *tpl,
				Name: *stackName,
			},
		}
	}

	serv := serv(cfg)

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

func serv(cfg Config) claws.Service {
	awsConfig := awsprov.Config{}

	if profile, ok := cfg.Parameters["Profile"]; ok && profile != "" {
		awsConfig.Profile = profile
	}

	if region, ok := cfg.Parameters["Region"]; ok && region != "" {
		awsConfig.Region = region
	}

	return claws.Service{
		Approver:      &cli.Approval{},
		Log:           log.New(os.Stderr, "", log.LstdFlags),
		CloudProvider: awsprov.New(awsConfig),
	}
}

func readConfigs(cfgFiles cfgFilesFlags) Config {
	mainConfig := Config{}

	if len(cfgFiles) == 0 {
		if _, err := os.Stat("Claws.toml"); err == nil {
			cfgFiles = cfgFilesFlags{"Claws.toml"}
		}
	}

	for _, cf := range cfgFiles {
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
