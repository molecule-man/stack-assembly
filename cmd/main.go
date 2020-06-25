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

	cmd.NormalizeAwscliParamsIfNeeded()

	err := cmd.RootCmd().Execute()

	if err != nil {
		switch {
		case errors.Is(err, commands.ErrNotRunnable), strings.HasPrefix(err.Error(), "unknown command"):
			if os.Getenv("STAS_SUPPRESS_CMD_NOT_FOUND_ERROR") != "yes" {
				console.Error(err.Error())
			}

			os.Exit(2)
		case errors.Is(err, commands.ErrAwsDropInArgParsingFailed):
			console.Error(err.Error())
			os.Exit(3)
		case strings.HasPrefix(err.Error(), "unknown flag"):
			console.Error(err.Error())
			os.Exit(4)
		case strings.HasPrefix(err.Error(), "invalid argument"):
			console.Error(err.Error())
			os.Exit(5)
		case errors.Is(err, commands.ErrInvalidInput):
			console.Error(err.Error())
			os.Exit(6)
		default:
			console.Error(err.Error())
			os.Exit(1)
		}
	}

	assembly.MustSucceed(err)
}
