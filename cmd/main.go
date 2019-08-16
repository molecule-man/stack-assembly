// Package main provides cmd stas application
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	assembly "github.com/molecule-man/stack-assembly"
	"github.com/molecule-man/stack-assembly/cli"
	"github.com/molecule-man/stack-assembly/cli/color"
	"github.com/molecule-man/stack-assembly/conf"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	yaml "gopkg.in/yaml.v2"
)

var cfg = conf.Config{
	Parameters:   map[string]string{},
	Tags:         map[string]string{},
	Capabilities: []string{},
}

var cfgFilesFlag = pflag.NewFlagSet("configs", pflag.ContinueOnError)
var cfgFiles []string
var nonInteractive bool

func main() {
	rootCmd := rootCmd()

	cfgFilesFlag.StringSliceVarP(&cfgFiles, "configs", "c", []string{},
		"Alternative config file(s). Default: stack-assembly.yaml")

	rootCmd.AddCommand(
		infoCmd(),
		syncCmd(),
		deployCmd(),
		diffCmd(),
		deleteCmd(),
		dumpConfigCmd(),
	)

	assembly.MustSucceed(rootCmd.Execute())
}

func rootCmd() *cobra.Command {
	var nocolor bool

	defaultProfile := "default"
	if profile := os.Getenv("AWS_PROFILE"); profile != "" {
		defaultProfile = profile
	}

	rootCmd := &cobra.Command{
		Use: "stas <stack name> <template path>",
	}
	rootCmd.PersistentFlags().StringVarP(&cfg.Settings.Aws.Profile, "profile", "p", defaultProfile, "AWS named profile")
	rootCmd.PersistentFlags().StringVarP(&cfg.Settings.Aws.Region, "region", "r", os.Getenv("AWS_REGION"), "AWS region")

	// rootCmd.PersistentFlags().StringSliceVarP(&cfgFiles, "configs", "c", []string{},
	// 	"Alternative config file(s). Default: stack-assembly.yaml")
	rootCmd.PersistentFlags().BoolVar(&nocolor, "nocolor", false,
		"Disables color output")
	rootCmd.PersistentFlags().BoolVarP(&nonInteractive, "no-interaction", "n", false,
		"Do not ask any interactive questions")

	rootCmd.PersistentFlags().StringToStringVarP(&cfg.Parameters, "var", "v", map[string]string{},
		"Additional variables to use as parameters in config.\nExample: -v myParam=someValue")

	cobra.OnInitialize(func() {
		color.NoColor = nocolor
	})

	return rootCmd
}

func infoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show info about the stacks",
		Run: func(cmd *cobra.Command, args []string) {
			err := conf.LoadConfig(cfgFiles, &cfg)
			assembly.MustSucceed(err)

			assembly.InfoAll(cfg)
		},
	}

	cmd.Flags().AddFlagSet(cfgFilesFlag)
	return cmd
}

func deployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy <stack name> <template path>",
		Args:  cobra.ExactArgs(2),
		Short: "Deploys single cloudformation template",
		Run: func(cmd *cobra.Command, args []string) {
			cfg.Name = args[0]
			cfg.Path = args[1]

			assembly.MustSucceed(conf.InitConfig(&cfg))
			assembly.Sync(cfg, nonInteractive)
		},
	}

	cmd.Flags().StringSliceVar(&cfg.Capabilities, "capabilities", cfg.Capabilities,
		"A list of capabilities that you must specify before AWS\nCloudformation can create certain stacks. E.g. CAPABILITY_IAM")
	cmd.Flags().StringToStringVar(&cfg.Tags, "tags", cfg.Tags, "A list of tags to associate with the stack that is deployed")

	return cmd
}

func syncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync [<ID> [<ID> ...]]",
		Short: "Deploy stacks using the config file(s)",
		Long: `Creates or updates stacks specified in the config file(s).

By default sync command deploys all the stacks described in the config file(s).
To deploy a particular stack, ID argument has to be provided. ID is an
identifier of a stack within the config file. For example, ID is tpl1 in the
following yaml config:

    stacks:
      tpl1: # <--- this is ID
        name: mystack
        path: path/to/tpl.json

The config can be nested:
    stacks:
      parent_tpl:
        name: my-parent-stack
        path: path/to/tpl.json
        stacks:
          child_tpl: # <--- this is ID of the stack we want to deploy
            name: my-child-stack
            path: path/to/tpl.json

In this case specifying ID of only wanted stack is not enough all the parent IDs
have to be specified as well:

  stas sync parent_tpl child_tpl`,

		Run: func(cmd *cobra.Command, args []string) {
			err := conf.LoadConfig(cfgFiles, &cfg)
			assembly.MustSucceed(err)

			for _, id := range args {
				stack, ok := cfg.Stacks[id]
				if !ok {
					foundIds := make([]string, 0, len(cfg.Stacks))
					for id := range cfg.Stacks {
						foundIds = append(foundIds, id)
					}

					assembly.MustSucceed(fmt.Errorf("ID %s is not found in the config. Found IDs: %v", id, foundIds))
				}

				cfg = stack
			}

			assembly.Sync(cfg, nonInteractive)
		},
	}

	cmd.Flags().AddFlagSet(cfgFilesFlag)
	return cmd
}

func diffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show diff of the stacks to be deployed",
		Run: func(cmd *cobra.Command, args []string) {
			err := conf.LoadConfig(cfgFiles, &cfg)
			assembly.MustSucceed(err)

			assembly.Diff(cfg)
		},
	}

	cmd.Flags().AddFlagSet(cfgFilesFlag)
	return cmd
}

func deleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Deletes deployed stacks",
		Run: func(cmd *cobra.Command, args []string) {
			err := conf.LoadConfig(cfgFiles, &cfg)
			assembly.MustSucceed(err)

			assembly.Delete(cfg, nonInteractive)
		},
	}

	cmd.Flags().AddFlagSet(cfgFilesFlag)
	return cmd
}

func dumpConfigCmd() *cobra.Command {
	var format string
	dumpCmd := &cobra.Command{
		Use:   "dump-config",
		Short: "Dump loaded config into stdout",
		Run: func(cmd *cobra.Command, _ []string) {
			err := conf.LoadConfig(cfgFiles, &cfg)
			assembly.MustSucceed(err)

			dumpCfg(format)
		},
	}
	dumpCmd.Flags().StringVarP(&format, "format", "f", "yaml", "One of: yaml, toml, json")
	dumpCmd.Flags().AddFlagSet(cfgFilesFlag)

	return dumpCmd
}

func dumpCfg(format string) {
	out := cli.Output

	switch format {
	case "yaml", "yml":
		assembly.MustSucceed(yaml.NewEncoder(out).Encode(cfg))
	case "json":
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		assembly.MustSucceed(enc.Encode(cfg))
	case "toml":
		assembly.MustSucceed(toml.NewEncoder(out).Encode(cfg))
	default:
		assembly.Terminate("unknown format: " + format)
	}
}
