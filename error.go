package assembly

import (
	"os"

	"github.com/molecule-man/stack-assembly/cli"
)

func MustSucceed(err error) {
	if err != nil {
		Terminate(err.Error())
	}
}

func Terminate(msg string) {
	cli := cli.CLI{Errorer: os.Stderr}
	cli.Error(msg)
	os.Exit(1)
}
