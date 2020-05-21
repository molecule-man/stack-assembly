package assembly

import (
	"github.com/molecule-man/stack-assembly/awscf"
	"github.com/molecule-man/stack-assembly/conf"
)

// TODO fix diff for bodies larger than 50k (need to upload to s3 first)

func (sa SA) Diff(cfg conf.Config) error {
	for _, childCfg := range cfg.Stacks {
		err := sa.Diff(childCfg)
		if err != nil {
			return err
		}
	}

	if cfg.Body == "" {
		return nil
	}

	diff, err := awscf.ChSetDiff{Color: sa.cli.Color}.Diff(cfg.ChangeSet())
	if err != nil {
		return err
	}

	sa.cli.Print(diff)

	return nil
}
