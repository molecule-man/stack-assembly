package assembly

import (
	"github.com/molecule-man/stack-assembly/awscf"
	"github.com/molecule-man/stack-assembly/cli"
	"github.com/molecule-man/stack-assembly/conf"
)

func Diff(cfg conf.Config) {
	for _, childCfg := range cfg.Stacks {
		Diff(childCfg)
	}

	if cfg.Body == "" {
		return
	}

	diff, err := awscf.Diff(cfg.ChangeSet())
	MustSucceed(err)

	cli.Print(diff)
}
