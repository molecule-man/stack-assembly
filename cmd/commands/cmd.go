package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	assembly "github.com/molecule-man/stack-assembly"
	"github.com/molecule-man/stack-assembly/cli"
	"github.com/molecule-man/stack-assembly/conf"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

const (
	wrapFlagLen = 60
)

type Commands struct {
	SA             *assembly.SA
	Cli            *cli.CLI
	CfgLoader      *conf.Loader
	AWSCommandsCfg struct {
		SA     *assembly.SA
		output *string
	}
	NonInteractive *bool

	cfg      *conf.Config
	origArgs []string
}

func (c *Commands) RootCmd() *cobra.Command {
	if c.NonInteractive == nil {
		nonInteractive := false
		c.NonInteractive = &nonInteractive
	}

	out := "text"
	c.AWSCommandsCfg.output = &out

	c.cfg = &conf.Config{
		Parameters:   map[string]string{},
		Tags:         map[string]string{},
		Capabilities: []string{},
	}

	defaultProfile := "default"
	if profile := os.Getenv("AWS_PROFILE"); profile != "" {
		defaultProfile = profile
	}

	rootCmd := &cobra.Command{
		Use:           "stas <stack name> <template path>",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	rootCmd.PersistentFlags().StringVarP(&c.cfg.Settings.Aws.Profile, "profile", "p", defaultProfile, "AWS named profile")
	rootCmd.PersistentFlags().StringVarP(&c.cfg.Settings.Aws.Region, "region", "r", os.Getenv("AWS_REGION"), "AWS region")
	rootCmd.PersistentFlags().StringVar(&c.cfg.Settings.Aws.Endpoint, "endpoint-url", "", "AWS endpoint url")

	rootCmd.PersistentFlags().BoolVar(&c.Cli.Color.Disabled, "nocolor", false,
		"Disables color output")
	rootCmd.PersistentFlags().BoolVarP(c.NonInteractive, "no-interaction", "n", *c.NonInteractive,
		"Do not ask any interactive questions")

	rootCmd.PersistentFlags().StringToStringVarP(&c.cfg.Parameters, "var", "v", map[string]string{},
		"Additional variables to use as parameters in config.\nExample: -v myParam=someValue")

	rootCmd.AddCommand(
		c.infoCmd(),
		c.syncCmd(),
		c.deployCmd(),
		c.diffCmd(),
		c.deleteCmd(),
		c.dumpConfigCmd(),
		c.cloudformationCmd(),
	)

	return rootCmd
}

func (c Commands) infoCmd() *cobra.Command {
	cfgFiles := []string{}
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show info about the stacks",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.CfgLoader.LoadConfig(cfgFiles, c.cfg); err != nil {
				return err
			}

			return c.SA.InfoAll(*c.cfg)
		},
	}

	cmd.Flags().StringVar(&c.cfg.Name, "stack-name", "", flagDescription("Stack name"))

	addConfigFlag(cmd, &cfgFiles)

	return cmd
}

func (c Commands) deployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy <stack name> <template path>",
		Args:  cobra.ExactArgs(2),
		Short: "Deploys single cloudformation template",
		RunE: func(cmd *cobra.Command, args []string) error {
			c.cfg.Name = args[0]
			c.cfg.Path = args[1]

			if err := c.CfgLoader.InitConfig(c.cfg); err != nil {
				return err
			}

			_, err := c.SA.Sync(*c.cfg, *c.NonInteractive)
			return err
		},
	}

	cmd.Flags().StringSliceVar(&c.cfg.Capabilities, "capabilities", c.cfg.Capabilities,
		"A list of capabilities that you must specify before AWS\nCloudformation can create certain stacks. E.g. CAPABILITY_IAM")
	cmd.Flags().StringToStringVar(&c.cfg.Tags, "tags", c.cfg.Tags, "A list of tags to associate with the stack that is deployed")

	return cmd
}

func (c Commands) syncCmd() *cobra.Command {
	cfgFiles := []string{}
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

		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.CfgLoader.LoadConfig(cfgFiles, c.cfg); err != nil {
				return err
			}

			for _, id := range args {
				stack, ok := c.cfg.Stacks[id]
				if !ok {
					foundIds := make([]string, 0, len(c.cfg.Stacks))
					for id := range c.cfg.Stacks {
						foundIds = append(foundIds, id)
					}

					assembly.MustSucceed(fmt.Errorf("ID %s is not found in the config. Found IDs: %v", id, foundIds))
				}

				*c.cfg = stack
			}

			_, err := c.SA.Sync(*c.cfg, *c.NonInteractive)
			return err
		},
	}

	addConfigFlag(cmd, &cfgFiles)

	return cmd
}

func (c Commands) diffCmd() *cobra.Command {
	cfgFiles := []string{}
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show diff of the stacks to be deployed",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.CfgLoader.LoadConfig(cfgFiles, c.cfg); err != nil {
				return err
			}
			return c.SA.Diff(*c.cfg)
		},
	}

	addConfigFlag(cmd, &cfgFiles)

	return cmd
}

func (c Commands) deleteCmd() *cobra.Command {
	cfgFiles := []string{}
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Deletes deployed stacks",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.CfgLoader.LoadConfig(cfgFiles, c.cfg); err != nil {
				return err
			}

			return c.SA.Delete(*c.cfg, *c.NonInteractive)
		},
	}

	addConfigFlag(cmd, &cfgFiles)

	return cmd
}

func (c Commands) dumpConfigCmd() *cobra.Command {
	var format string

	cfgFiles := []string{}

	dumpCmd := &cobra.Command{
		Use:   "dump-config",
		Short: "Dump loaded config into stdout",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := c.CfgLoader.LoadConfig(cfgFiles, c.cfg); err != nil {
				return err
			}

			c.dumpCfg(format)
			return nil
		},
	}
	dumpCmd.Flags().StringVarP(&format, "format", "f", "yaml", "One of: yaml, toml, json")
	addConfigFlag(dumpCmd, &cfgFiles)

	return dumpCmd
}

func (c Commands) cloudformationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cloudformation",
		Short: "Drop-in replacement of several aws cloudformation commands",

		// next configs are here to prevent showing help when
		// `cloudformation bullshit` command requested.  see also
		// https://github.com/spf13/cobra/issues/582
		DisableFlagParsing:    true,
		DisableFlagsInUseLine: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return ErrNotRunnable
		},
	}

	cmd.AddCommand(
		c.cfDeployCmd(),
		c.cfCreateCmd(),
		c.cfUpdateCmd(),
	)

	return cmd
}

func (c Commands) cfDeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Drop-in replacement of `aws cloudformation deploy` command",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.CfgLoader.InitConfig(c.cfg); err != nil {
				return err
			}
			_, err := c.AWSCommandsCfg.SA.Sync(*c.cfg, *c.NonInteractive)
			return err
		},
	}

	cmd.Flags().StringVar(&c.cfg.Path, "template-file", "",
		flagDescription("Path to cloudformation template file"))

	cmd.Flags().StringVar(&c.cfg.Settings.S3Settings.BucketName, "s3-bucket", "", flagDescription(
		"The name of the S3 bucket where this command uploads your CloudFormation template.",
		" This is required the deployments of templates sized greater than 51,200 bytes"))

	cmd.Flags().StringVar(&c.cfg.Settings.S3Settings.Prefix, "s3-prefix", "", flagDescription(
		"A prefix name that the command adds to the artifacts' name when it uploads them to the S3 bucket.",
		" The prefix name is a path name (folder name) for the S3 bucket."))

	cmd.Flags().StringVar(&c.cfg.Settings.S3Settings.KMSKeyID, "kms-key-id", "", flagDescription(
		"The ID of an AWS KMS key that the command uses to encrypt artifacts that are at rest in the S3 bucket"))

	cmd.Flags().StringToStringVar(&c.cfg.Parameters, "parameter-overrides", map[string]string{},
		flagDescription("A list of parameter structures that specify input parameters for your stack template"))

	cmd.Flags().Bool("fail-on-empty-changeset", false, "This flag is ignored")
	cmd.Flags().Bool("no-fail-on-empty-changeset", true, "This flag is ignored")
	cmd.Flags().Bool("no-execute-changeset", false, "This flag is ignored")
	cmd.Flags().Bool("force-upload", false, "This flag is ignored")

	c.cfSharedFlags(cmd)

	assembly.MustSucceed(cmd.MarkFlagRequired("stack-name"))
	assembly.MustSucceed(cmd.MarkFlagRequired("template-file"))

	return cmd
}

func (c Commands) cfCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-stack",
		Short: "Drop-in replacement of `aws cloudformation create-stack` command",
		Long: cli.WordWrap(80,
			"Creates a stack. Drop-in replacement of `aws cloudformation create-stack` command\n",
			"\n",
			"The following flags are not implemented.\n",
			" * --rollback-configuration\n",
			" * --disable-rollback\n",
			" * --no-disable-rollback\n",
			" * --timeout-in-minutes\n",
			" * --on-failure\n",
			" * --stack-policy-body\n",
			" * --stack-policy-url\n",
			" * --enable-termination-protection\n",
			" * --cli-input-json\n",
			" * --generate-cli-skeleton\n",
		),
	}
	cmd.RunE = c.cfCreateUpdateFunc(cmd)

	c.cfCreateUpdateFlags(cmd)

	assembly.MustSucceed(cmd.MarkFlagRequired("stack-name"))

	return cmd
}

func (c Commands) cfUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-stack",
		Short: "Drop-in replacement of `aws cloudformation update-stack` command",
		Long: cli.WordWrap(80,
			"Updates a stack. Drop-in replacement of `aws cloudformation update-stack` command\n",
			"\n",
			"The following flags are not implemented.\n",
			" * --stack-policy-during-update-body\n",
			" * --stack-policy-during-update-url\n",
			" * --rollback-configuration\n",
			" * --stack-policy-body\n",
			" * --stack-policy-url\n",
			" * --cli-input-json\n",
			" * --generate-cli-skeleton\n",
		),
	}
	cmd.RunE = c.cfCreateUpdateFunc(cmd)

	c.cfCreateUpdateFlags(cmd)

	cmd.Flags().BoolVar(
		&c.cfg.UsePreviousTemplate, "use-previous-template", false, flagDescription(
			"Reuse the existing template that is associated with the stack that you are updating."))

	cmd.Flags().Bool("no-use-previous-template", false, "This flag is ignored")

	assembly.MustSucceed(cmd.MarkFlagRequired("stack-name"))

	return cmd
}

func (c Commands) cfCreateUpdateFunc(cmd *cobra.Command) func(*cobra.Command, []string) error {
	params := []string{}
	cmd.Flags().StringArrayVar(&params, "parameters", []string{}, flagDescription("List of parameters"))

	return func(cmd *cobra.Command, args []string) error {
		var err error
		if c.cfg.Parameters, err = awsParamsToMap(params); err != nil {
			return err
		}

		isFilePrefix := "file://"
		if strings.HasPrefix(c.cfg.Body, isFilePrefix) {
			c.cfg.Path = c.cfg.Body[len(isFilePrefix):]
			c.cfg.Body = ""
		}

		if err = c.CfgLoader.InitConfig(c.cfg); err != nil {
			return err
		}

		stacks, err := c.AWSCommandsCfg.SA.Sync(*c.cfg, *c.NonInteractive)
		if err != nil {
			return err
		}

		info, err := stacks[0].Info()
		if err != nil {
			return err
		}

		if *c.AWSCommandsCfg.output == "json" {
			out := struct {
				StackID string `json:"StackId"`
			}{StackID: info.ID()}

			enc := json.NewEncoder(c.Cli.Writer)
			enc.SetIndent("", "    ")

			return enc.Encode(out)
		}

		c.Cli.Print(info.ID())

		return nil
	}
}

func (c Commands) cfCreateUpdateFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.cfg.Body, "template-body", "", flagDescription("Cloudformation template body"))
	cmd.Flags().StringVar(&c.cfg.URL, "template-url", "", flagDescription("Cloudformation template url (s3)"))

	cmd.Flags().StringSliceVar(
		&c.cfg.ResourceTypes, "resource-types", []string{}, flagDescription(
			"The template resource types that you have permissions",
			" to work with for this create stack action"))

	cmd.Flags().StringVar(
		&c.cfg.ClientToken, "client-request-token", "", flagDescription(
			"A unique identifier for this request. Specify this",
			" token if you plan to retry requests"))

	c.cfSharedFlags(cmd)
}

func (c Commands) cfSharedFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.cfg.Name, "stack-name", "", flagDescription("Stack name"))

	cmd.Flags().StringSliceVar(
		&c.cfg.NotificationARNs, "notification-arns", []string{}, flagDescription(
			"SNS topic ARNs that AWS CloudFormation associates ",
			"with the stack."))

	cmd.Flags().StringToStringVar(
		&c.cfg.Tags, "tags", map[string]string{}, flagDescription(
			"A list of tags to associate with the stack that is ",
			"created or updated"))

	cmd.Flags().StringSliceVar(
		&c.cfg.Capabilities, "capabilities", []string{}, flagDescription(
			"A list of capabilities that you must specify before ",
			"AWS Cloudformation can create certain stacks. E.g. ",
			"CAPABILITY_IAM"))

	cmd.Flags().StringVar(
		&c.cfg.RoleARN, "role-arn", "", flagDescription(
			"ARN of an IAM role that AWS CloudFormation assumes ",
			"when executing the change set"))

	*c.AWSCommandsCfg.output = "text"
	outFlag := EnumFlag{
		Val:   c.AWSCommandsCfg.output,
		Enums: []string{"text", "json"},
	}
	cmd.Flags().Var(&outFlag, "output", flagDescription(
		"The formatting style for command output. Either text or json"))
}

func (c Commands) dumpCfg(format string) {
	out := c.Cli.Writer

	switch format {
	case "yaml", "yml":
		assembly.MustSucceed(yaml.NewEncoder(out).Encode(c.cfg))
	case "json":
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		assembly.MustSucceed(enc.Encode(c.cfg))
	case "toml":
		assembly.MustSucceed(toml.NewEncoder(out).Encode(c.cfg))
	default:
		assembly.Terminate("unknown format: " + format)
	}
}

func addConfigFlag(cmd *cobra.Command, val *[]string) {
	cmd.Flags().StringSliceVarP(val, "configs", "c", []string{},
		"Alternative config file(s). Default: stack-assembly.yaml")
}

func flagDescription(text ...string) string {
	return cli.WordWrap(wrapFlagLen, text...)
}

type EnumFlag struct {
	Val   *string
	Enums []string
}

func (f *EnumFlag) Set(val string) error {
	for _, e := range f.Enums {
		if e == val {
			*f.Val = val
			return nil
		}
	}

	return fmt.Errorf("value '%s' is not supported. Supported values are: %v: %w", val, f.Enums, ErrInvalidInput)
}
func (f *EnumFlag) Type() string   { return "enum" }
func (f *EnumFlag) String() string { return *f.Val }

var ErrNotRunnable = errors.New("command is not runnable")
var ErrInvalidInput = errors.New("invalid input")
