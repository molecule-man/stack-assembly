// Package main provides cmd stas application
package main

import (
	"os"

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
	assembly.MustSucceed(cmd.RootCmd().Execute())
}
