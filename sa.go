package assembly

import "github.com/molecule-man/stack-assembly/cli"

type SA struct {
	cli *cli.CLI
}

func New(c *cli.CLI) *SA {
	return &SA{c}
}
