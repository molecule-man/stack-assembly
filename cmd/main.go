// Package main provides cmd stas application
package main

import (
	"os"
	"strings"

	assembly "github.com/molecule-man/stack-assembly"
	"github.com/molecule-man/stack-assembly/aws"
	"github.com/molecule-man/stack-assembly/cli"
	"github.com/molecule-man/stack-assembly/cmd/commands"
	"github.com/molecule-man/stack-assembly/conf"
)

func main() {
	cli := &cli.CLI{
		Reader:  os.Stdin,
		Writer:  os.Stdout,
		Errorer: os.Stderr,
	}

	cmd := commands.Commands{
		SA:  assembly.New(cli),
		Cli: cli,
		CfgLoader: conf.NewLoader(
			&conf.OsFS{},
			&aws.Provider{},
		),
	}

	cmd.InvokeAwscliIfNeeded()

	err := cmd.RootCmd().Execute()

	if err != nil && cmd.IsAwsDropIn() && commands.IsAwsDropInError(err) {
		yesNo, promptErr := cli.Fask(
			os.Stderr,
			"%s\nDo you want to execute corresponding aws cli command [y|n]: ",
			err.Error(),
		)
		assembly.MustSucceed(promptErr)

		yesNo = strings.ToLower(yesNo)

		if yesNo == "y" || yesNo == "yes" {
			cmd.InvokeAwscli()
			return
		}
	}

	assembly.MustSucceed(err)
}
