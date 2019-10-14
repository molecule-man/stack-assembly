package assembly

import (
	"github.com/molecule-man/stack-assembly/awscf"
	"github.com/molecule-man/stack-assembly/conf"
)

func (sa SA) Diff(cfg conf.Config) {
	for _, childCfg := range cfg.Stacks {
		sa.Diff(childCfg)
	}

	if cfg.Body == "" {
		return
	}

	diff, err := awscf.ChSetDiff{Color: sa.cli.Color}.Diff(cfg.ChangeSet())
	MustSucceed(err)

	sa.cli.Print(diff)
}
