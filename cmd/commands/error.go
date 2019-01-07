package commands

import (
	"os"

	"github.com/molecule-man/stack-assembly/cli"
)

func handleError(err error) {
	if err != nil {
		cli.Errorf("%s", err)
		os.Exit(1)
	}
}
