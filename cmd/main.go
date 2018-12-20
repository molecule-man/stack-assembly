// Package main provides cmd stas application
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"text/tabwriter"

	"github.com/molecule-man/stack-assembly/awsprov"
	"github.com/molecule-man/stack-assembly/cli"
	"github.com/molecule-man/stack-assembly/cmd/conf"
	"github.com/molecule-man/stack-assembly/stackassembly"
	"github.com/spf13/cobra"
)

func main() {
	var cfgFiles []string
	var stackName string

	rootCmd := &cobra.Command{
		Use: "stas",
	}
	rootCmd.PersistentFlags().StringArrayVarP(&cfgFiles, "configs", "f", []string{}, "CF configs")

	syncCmd := &cobra.Command{
		Use:   "sync [tpl]",
		Short: "Sync stacks",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				execSyncOneTpl(stackName, args[0])
			} else {
				sync(readConfigs(cfgFiles))
			}
		},
	}
	syncCmd.Flags().StringVarP(&stackName, "stack", "s", "", "Stack name")

	infoCmd := &cobra.Command{
		Use:   "info [stack name]",
		Short: "Show info about the stack",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfg := readConfigs(cfgFiles)
			serv := serv(cfg)

			for k, template := range cfg.Templates {

				if len(args) > 0 && args[0] != template.Name && args[0] != k {
					continue
				}

				tpl := stackassembly.StackTemplate{Name: template.Name, Params: template.Parameters}

				info, err := serv.Info(tpl, cfg.Parameters)

				if err != nil {
					log.Fatal(err)
				}

				displayStackInfo(info)
			}
		},
	}

	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(syncCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func displayStackInfo(info stackassembly.StackInfo) {

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

func execSyncOneTpl(stackName, tpl string) {
	cfg := conf.Config{}

	cfg.Templates = map[string]conf.TemplateConfig{
		stackName: {
			Path: tpl,
			Name: stackName,
		},
	}

	sync(cfg)
}

func sync(cfg conf.Config) {
	serv := serv(cfg)

	tpls := make(map[string]stackassembly.StackTemplate)

	for i, template := range cfg.Templates {
		tplBody, err := ioutil.ReadFile(template.Path)

		if err != nil {
			log.Fatal(err)
		}

		tpls[i] = stackassembly.StackTemplate{
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

func serv(cfg conf.Config) stackassembly.Service {
	awsConfig := awsprov.Config{}

	if profile, ok := cfg.Parameters["Profile"]; ok && profile != "" {
		awsConfig.Profile = profile
	}

	if region, ok := cfg.Parameters["Region"]; ok && region != "" {
		awsConfig.Region = region
	}

	return stackassembly.Service{
		Approver:      &cli.Approval{},
		Log:           log.New(os.Stderr, "", log.LstdFlags),
		CloudProvider: awsprov.New(awsConfig),
	}
}

func readConfigs(cfgFiles []string) conf.Config {
	return conf.LoadConfig(cfgFiles)
}
