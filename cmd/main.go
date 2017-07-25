// Package main provides cmd claws application
package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/molecule-man/claws/claws"
	"github.com/molecule-man/claws/cli"
)

func main() {
	tpl := flag.String("tpl", "", "CF tpl")
	cfgFile := flag.String("cfg", "", "CF tpl")
	stackName := flag.String("stack", "", "Stack name")

	flag.Parse()

	mainConfig := claws.Config{}
	configFiles := make([]string, 0)

	if _, err := os.Stat("Claws.toml"); err == nil {
		configFiles = append(configFiles, "Claws.toml")
	}
	if cfgFile != nil && *cfgFile != "" {
		for _, c := range strings.Split(*cfgFile, " ") {
			configFiles = append(configFiles, c)
		}
	}

	for _, cf := range configFiles {
		cfg := claws.Config{}
		_, err := toml.DecodeFile(cf, &cfg)

		if err != nil {
			log.Fatal(err)
		}

		mainConfig.Merge(cfg)
	}

	if tpl != nil && *tpl != "" {

		mainConfig.Templates = map[string]claws.TemplateConfig{
			"tpl": {
				Path: *tpl,
				Name: stackName,
			},
		}
	}

	serv := claws.Service{
		Approver:        &cli.Approval{},
		Log:             log.New(os.Stderr, "", log.LstdFlags),
		ChangePresenter: &cli.ChangeTable{},
	}

	for _, template := range mainConfig.Templates {
		tplBody, err := ioutil.ReadFile(template.Path)

		if err != nil {
			log.Fatal(err)
		}

		err = serv.Sync(*template.Name, string(tplBody), mainConfig.Parameters)
		if err != nil {
			log.Fatal(err)
		}
	}
}
