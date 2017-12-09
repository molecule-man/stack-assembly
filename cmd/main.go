// Package main provides cmd claws application
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/tabwriter"

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
	flag.Var(&cfgFiles, "f", "CF configs")

	tpl := flag.String("tpl", "", "CF tpl")
	stackName := flag.String("stack", "", "Stack name")
	infoMode := flag.Bool("i", false, "Info")

	flag.Parse()

	switch {
	case *infoMode:
		execInfo(readConfigs(cfgFiles))
	case tpl != nil && *tpl != "":
		execSyncOneTpl(*stackName, *tpl)
	default:
		sync(readConfigs(cfgFiles))
	}
}

func execInfo(cfg Config) {
	serv := serv(cfg)

	for _, template := range cfg.Templates {
		tpl := claws.StackTemplate{Name: template.Name, Params: template.Parameters}

		info, err := serv.Info(tpl, cfg.Parameters)

		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("===================")
		fmt.Printf("STACK: %s\n", info.Name)
		fmt.Println("===================")
		fmt.Println("")

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
		fmt.Println("==== RESOURCES ====")
		for _, res := range info.Resources {
			fmt.Fprintf(w, "%s\t%s\t%s\n", res.LogicalID, res.PhysicalID, res.Status)
		}
		w.Flush()
		fmt.Println("")

		fmt.Println("==== OUTPUTS ====")
		for _, out := range info.Outputs {
			fmt.Fprintf(w, "%s\t%s\t%s\n", out.Key, out.Value, out.ExportName)
		}
		w.Flush()
		fmt.Println("")

		fmt.Println("==== EVENTS ====")
		for _, e := range info.Events[:10] {
			fmt.Fprintf(w, "[%v]\t%s\t%s\t%s\t%s\n", e.Timestamp, e.ResourceType, e.LogicalResourceID, e.Status, e.StatusReason)
		}
		w.Flush()
		fmt.Print("\n\n")
	}
}

func execSyncOneTpl(stackName, tpl string) {
	cfg := Config{}

	cfg.Templates = map[string]TemplateConfig{
		stackName: {
			Path: tpl,
			Name: stackName,
		},
	}

	sync(cfg)
}

func sync(cfg Config) {
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
