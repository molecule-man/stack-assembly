// Package main provides cmd stas application
package main

import (
	"errors"
	"log"
	"os"
	"os/exec"

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

	if len(os.Args) > 1 && os.Args[1] == "--aws" {
		os.Args = append(os.Args[:1], os.Args[2:]...)
		_, _, err := cmd.RootCmd().Find(os.Args[1:])

		if err != nil {
			c := exec.Command("aws", os.Args[1:]...)
			c.Env = os.Environ()
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr

			if execErr := c.Run(); execErr != nil {
				var exitErr *exec.ExitError
				if errors.As(execErr, &exitErr) {
					os.Exit(exitErr.ExitCode())
				} else {
					log.Fatalf("aws exited unexpectedly: %v", execErr)
				}
			}

			return
		}
	}

	assembly.MustSucceed(cmd.RootCmd().Execute())
}
