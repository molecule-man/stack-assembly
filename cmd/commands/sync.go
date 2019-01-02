package commands

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/fatih/color"
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
	cfg := conf.Config{}

	cfg.Stacks = map[string]stackassembly.StackConfig{
		stackName: {
			Path: tpl,
			Name: stackName,
		},
	}

	sync(cfg, nonInteractive)
}

func sync(cfg conf.Config, nonInteractive bool) {
	aws := conf.Aws(cfg)

	stacks, err := cfg.GetStacks()
	handleError(err)
	err = stackassembly.SortStacksByExecOrder(stacks)
	handleError(err)

	logger := log.New(os.Stderr, "", log.LstdFlags)

	for _, stack := range stacks {
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

			if !nonInteractive {
				continueSync := false
				for !continueSync {
					prompt([]promptCmd{
						{
							description:   "[s]ync",
							triggerInputs: []string{"s", "sync"},
							action: func() {
								continueSync = true
							},
						},
						{
							description:   "[d]iff",
							triggerInputs: []string{"d", "diff"},
							action: func() {
								diffS := stackassembly.DiffService{
									Dp: aws,
								}

								diff, derr := diffS.Diff(stack)
								handleError(derr)

								fmt.Println(diff)
							},
						},
						{
							description:   "[q]uit",
							triggerInputs: []string{"q", "quit"},
							action: func() {
								print("Interrupted by user")
								handleError(errors.New("sync is cancelled"))
							},
						},
					})
				}
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

		green := color.New(color.FgGreen)
		cyan := color.New(color.FgCyan)
		boldRed := color.New(color.FgRed, color.Bold)

		for _, c := range changes {
			t.Row()

			switch strings.ToLower(c.Action) {
			case "add":
				t.ColorizedCell(c.Action, green)
			case "remove":
				t.ColorizedCell(c.Action, boldRed)
			default:
				t.ColorizedCell(c.Action, cyan)
			}

			t.Cell(c.ResourceType)
			t.Cell(c.LogicalResourceID)

			col := green
			if c.ReplacementNeeded {
				col = boldRed
			}
			t.ColorizedCell(fmt.Sprintf("%t", c.ReplacementNeeded), col)
		}

		fmt.Println(t.Render())
	}
}

type promptCmd struct {
	triggerInputs []string
	description   string
	action        func()
}

func prompt(commands []promptCmd) {
	fmt.Println("*** Commands ***")
	for _, c := range commands {
		fmt.Println("  " + c.description)
	}
	fmt.Print("What now> ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')

	if err != nil {
		handleError(fmt.Errorf("reading user input failed with err: %v", err))
	}

	response = strings.TrimSpace(response)

	for _, c := range commands {
		for _, inp := range c.triggerInputs {
			if inp == response {
				c.action()
				return
			}
		}
	}

	fmt.Printf("Command %s is not known\n", response)
}
