package commands

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/molecule-man/stack-assembly/cli"
	"github.com/molecule-man/stack-assembly/cmd/conf"
	"github.com/molecule-man/stack-assembly/stackassembly"
	"github.com/spf13/cobra"
)

func syncCmd() *cobra.Command {
	var stackName string
	var nonInteractive bool

	syncCmd := &cobra.Command{
		Use:   "sync [stack]",
		Short: "Sync stacks",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfgFiles, err := cmd.Parent().PersistentFlags().GetStringSlice("configs")
			handleError(err)

			if len(args) > 0 {
				execSyncOneTpl(stackName, args[0], nonInteractive)
			} else {
				cfg, err := conf.LoadConfig(cfgFiles)
				handleError(err)
				sync(cfg, nonInteractive)
			}
		},
	}
	syncCmd.Flags().StringVarP(&stackName, "stack", "s", "", "Stack name")
	syncCmd.Flags().BoolVarP(&nonInteractive, "no-interaction", "n", false, "Do not ask any interactive questions")

	return syncCmd
}

func execSyncOneTpl(stackName, tpl string, nonInteractive bool) {
	cfg := stackassembly.Config{}

	cfg.Stacks = map[string]stackassembly.StackConfig{
		stackName: {
			Path: tpl,
			Name: stackName,
		},
	}

	sync(cfg, nonInteractive)
}

func sync(cfg stackassembly.Config, nonInteractive bool) {
	aws := conf.Aws(cfg)

	for i, stack := range cfg.Stacks {
		tplBody, err := ioutil.ReadFile(stack.Path)
		handleError(err)

		stack.Body = string(tplBody)
		cfg.Stacks[i] = stack
	}

	ordered, err := stackassembly.StacksSortedByExecOrder(cfg)
	handleError(err)

	logger := log.New(os.Stderr, "", log.LstdFlags)

	for _, stack := range ordered {
		stack := stack // pin
		print := func(msg string, args ...interface{}) {
			logger.Print(fmt.Sprintf(fmt.Sprintf("[%s] %s", stack.Name, msg), args...))
		}

		print("Syncing template")

		chSet, err := stackassembly.New(aws, stack,
			stackassembly.WithEventSubscriber(func(e stackassembly.StackEvent) {
				print("[%s] [%s] [%s] %s", e.ResourceType, e.Status, e.LogicalResourceID, e.StatusReason)
			}),
		)

		if err == stackassembly.ErrNoChange {
			print("No changes to be synced")
		} else {
			handleError(err)

			print("Change set is created: %s", chSet.ID)

			showChanges(chSet.Changes)

			if !nonInteractive && !askConfirmation() {
				handleError(errors.New("sync is cancelled"))
			}

			err = chSet.Exec()
			handleError(err)

			print("Sync is finished")
		}

		for _, r := range stack.Blocked {
			print("Blocking resource %s", r)
			err := aws.BlockResource(stack.Name, r)

			handleError(err)
		}
	}
}

func showChanges(changes []stackassembly.Change) {
	if len(changes) > 0 {
		t := cli.NewTable()
		t.Header().Cell("Action").Cell("ResourceType").Cell("Resource ID").Cell("Replacement needed")

		for _, c := range changes {
			t.Row().
				Cell(c.Action).
				Cell(c.ResourceType).
				Cell(c.LogicalResourceID).
				Cell(fmt.Sprintf("%t", c.ReplacementNeeded))
		}

		fmt.Println(t.Render())
	}
}

func askConfirmation() bool {
	fmt.Print("Continue? [Y/n] ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')

	if err != nil {
		log.Fatalf("Reading user input failed with err: %v", err)
	}

	response = strings.Trim(response, " \n")

	for _, okayResponse := range []string{"", "y", "Y", "yes", "Yes", "YES"} {
		if response == okayResponse {
			return true
		}
	}

	fmt.Println("Interrupted by user")
	return false
}
