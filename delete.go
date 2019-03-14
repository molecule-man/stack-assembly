package assembly

import (
	"fmt"

	"github.com/molecule-man/stack-assembly/cli"
	"github.com/molecule-man/stack-assembly/cli/color"
	"github.com/molecule-man/stack-assembly/conf"
)

func Delete(cfg conf.Config, nonInteractive bool) {
	action := &deleteAction{nonInteractive}
	action.delete(cfg)
}

type deleteAction struct {
	nonInteractive bool
}

func (a *deleteAction) delete(cfg conf.Config) {
	ss, err := cfg.StackConfigsSortedByExecOrder()
	MustSucceed(err)

	// reverse order of stack configs
	for i, j := 0, len(ss)-1; i < j; i, j = i+1, j-1 {
		ss[i], ss[j] = ss[j], ss[i]
	}

	for _, s := range ss {
		Delete(s, a.nonInteractive)
	}

	if cfg.Name == "" {
		return
	}

	logger := cli.PrefixedLogger(fmt.Sprintf("[%s] ", cfg.Name))

	stack := cfg.Stack()

	exists, err := stack.Exists()
	MustSucceed(err)

	if !exists {
		logger.Info("Stack doesn't exist")
		return
	}

	logger.Warnf("Stack %s is about to be deleted", cfg.Name)

	skip := false

	if !a.nonInteractive {
		continueDelete := false
		for !continueDelete && !skip {
			err = cli.Prompt([]cli.PromptCmd{{
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
					Info(stack)
				},
			}, {
				Description:   "[s]kip",
				TriggerInputs: []string{"s", "skip"},
				Action: func() {
					skip = true
				},
			}, {
				Description:   "[q]uit",
				TriggerInputs: []string{"q", "quit"},
				Action: func() {
					cli.Error("Interrupted by user")
					Terminate("deletion is cancelled")
				},
			}})
			if err != cli.ErrPromptCommandIsNotKnown {
				MustSucceed(err)
			}
		}
	}

	if skip {
		return
	}

	err = stack.Delete()
	MustSucceed(err)
	logger.Print(color.Success("Stack is deleted successfully"))
}
