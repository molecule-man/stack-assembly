package assembly

import (
	"github.com/molecule-man/stack-assembly/awscf"
	"github.com/molecule-man/stack-assembly/conf"
)

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

	cs := cfg.ChangeSet()

	defer func() {
		if closeErr := cs.Close(); closeErr != nil {
			sa.cli.Warnf("Error while cleaning up: %s", closeErr.Error())
		}
	}()

	diff, err := awscf.ChSetDiff{Color: sa.cli.Color}.Diff(cs)
	if err != nil {
		return err
	}

	sa.cli.Print(diff)

	return nil
}
