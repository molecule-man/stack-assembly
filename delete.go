package assembly

import (
	"errors"
	"fmt"

	"github.com/molecule-man/stack-assembly/awscf"
	"github.com/molecule-man/stack-assembly/cli"
	"github.com/molecule-man/stack-assembly/conf"
)

func (sa *SA) Delete(cfg conf.Config, nonInteractive bool) error {
	action := &deleteAction{sa, sa.cli, nonInteractive}
	return action.delete(cfg)
}

type deleteAction struct {
	sa             *SA
	cli            *cli.CLI
	nonInteractive bool
}

func (a *deleteAction) delete(cfg conf.Config) error {
	ss, err := cfg.StackConfigsSortedByExecOrder()
	if err != nil {
		return err
	}

	// reverse order of stack configs
	for i, j := 0, len(ss)-1; i < j; i, j = i+1, j-1 {
		ss[i], ss[j] = ss[j], ss[i]
	}

	for _, s := range ss {
		nestedErr := a.delete(s)
		if nestedErr != nil {
			return nestedErr
		}
	}

	if cfg.Name == "" {
		return nil
	}

	logger := a.cli.PrefixedLogger(fmt.Sprintf("[%s] ", cfg.Name))

	stack := cfg.Stack()

	exists, err := stack.Exists()
	MustSucceed(err)

	if !exists {
		logger.Info("Stack doesn't exist")
		return nil
	}

	logger.Warnf("Stack %s is about to be deleted", cfg.Name)

	err = a.ask(stack)

	if errors.Is(err, errSkipDelete) {
		return nil
	}

	if err != nil {
		return err
	}

	err = stack.Delete()
	if err != nil {
		return err
	}

	logger.Print(a.cli.Color.Success("Stack is deleted successfully"))

	return nil
}

func (a *deleteAction) ask(stack *awscf.Stack) error {
	if a.nonInteractive {
		return nil
	}

	continueDelete := false
	for !continueDelete {
		var actionErr error

		err := a.cli.Prompt([]cli.PromptCmd{{
			Description:   "[d]elete",
			TriggerInputs: []string{"d", "delete"},
			Action: func() {
				continueDelete = true
			},
		}, {
			Description:   "[a]ll (delete all without asking again)",
			TriggerInputs: []string{"a", "all"},
			Action: func() {
				a.nonInteractive = true
				continueDelete = true
			},
		}, {
			Description:   "[i]nfo (show stack info)",
			TriggerInputs: []string{"i", "info"},
			Action: func() {
				actionErr = a.sa.Info(stack)
			},
		}, {
			Description:   "[s]kip",
			TriggerInputs: []string{"s", "skip"},
			Action: func() {
				actionErr = errSkipDelete
			},
		}, {
			Description:   "[q]uit",
			TriggerInputs: []string{"q", "quit"},
			Action: func() {
				a.cli.Error("Interrupted by user")
				actionErr = errors.New("deletion is canceled")
			},
		}})
		if err != nil && !errors.Is(err, cli.ErrPromptCommandIsNotKnown) {
			return err
		}

		if actionErr != nil {
			return actionErr
		}
	}

	return nil
}

var errSkipDelete = errors.New("deletion skipped")
