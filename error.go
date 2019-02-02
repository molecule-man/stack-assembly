package assembly

import (
	"os"

	"github.com/molecule-man/stack-assembly/cli"
)

func MustSucceed(err error) {
	if err != nil {
		cli.Errorf("%s", err)
		os.Exit(1)
	}
}
