// Package main provides cmd stas application
package main

import (
	"errors"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	assembly "github.com/molecule-man/stack-assembly"
	"github.com/molecule-man/stack-assembly/aws"
	"github.com/molecule-man/stack-assembly/cli"
	"github.com/molecule-man/stack-assembly/cmd/commands"
	"github.com/molecule-man/stack-assembly/conf"
)

func main() {
	console := &cli.CLI{
		Reader:  os.Stdin,
		Writer:  os.Stdout,
		Errorer: os.Stderr,
	}

	nonInteractive := !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd())

	cmd := commands.Commands{
		SA:  assembly.New(console),
		Cli: console,
		CfgLoader: conf.NewLoader(
			&conf.OsFS{},
			&aws.Provider{},
		),
		NonInteractive: &nonInteractive,
	}

	cmd.AWSCommandsCfg.SA = assembly.New(&cli.CLI{
		Reader:  os.Stdin,
		Writer:  os.Stderr,
		Errorer: os.Stderr,
	})

	childCmd, _, err := cmd.RootCmd().Find(os.Args[1:])

	if err != nil || !childCmd.Runnable() {
		if os.Getenv("STAS_SUPPRESS_CMD_NOT_FOUND_ERROR") != "yes" {
			errMsg := "the command is not runnable"
			if err != nil {
				errMsg = err.Error()
			}

			console.Error(errMsg)
		}

		os.Exit(2)
	}

	cmd.NormalizeAwscliParamsIfNeeded()

	err = cmd.RootCmd().Execute()

	if err != nil {
		status := 1

		console.Error(err.Error())

		switch {
		case errors.Is(err, commands.ErrAwsDropInArgParsingFailed):
			status = 3
		case strings.HasPrefix(err.Error(), "unknown flag"):
			status = 4
		case strings.HasPrefix(err.Error(), "invalid argument"):
			status = 5
		}

		os.Exit(status)
	}

	assembly.MustSucceed(err)
}
