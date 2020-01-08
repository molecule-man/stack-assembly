// Package main provides cmd stas application
package main

import (
	"errors"
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

	childCmd, _, err := cmd.RootCmd().Find(os.Args[1:])

	if err != nil || !childCmd.Runnable() {
		if os.Getenv("STAS_SUPPRESS_CMD_NOT_FOUND_ERROR") != "yes" {
			errMsg := "the command is not runnable"
			if err != nil {
				errMsg = err.Error()
			}

			cli.Error(errMsg)
		}

		os.Exit(4)
	}

	cmd.NormalizeAwscliParamsIfNeeded()

	err = cmd.RootCmd().Execute()

	if err != nil {
		status := 1

		cli.Error(err.Error())

		switch {
		case errors.Is(err, commands.ErrAwsDropIn):
			status = 2
		case strings.HasPrefix(err.Error(), "unknown flag"):
			status = 3
		}

		os.Exit(status)
	}

	assembly.MustSucceed(err)
}
