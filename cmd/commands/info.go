package commands

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/molecule-man/stack-assembly/awsprov"
	"github.com/molecule-man/stack-assembly/cmd/conf"
	"github.com/molecule-man/stack-assembly/stackassembly"
	"github.com/spf13/cobra"
)

func infoCmd() *cobra.Command {
	infoCmd := &cobra.Command{
		Use:   "info [stack name]",
		Short: "Show info about the stack",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfgFiles, err := cmd.Parent().PersistentFlags().GetStringSlice("configs")
			handleError(err)

			cfg, err := conf.LoadConfig(cfgFiles)
			handleError(err)

			printer := newInfoPrinter(os.Stdout, conf.Aws(cfg))

			stacks, err := cfg.GetStacks()
			handleError(err)

			for _, stack := range stacks {
				if len(args) > 0 && args[0] != stack.Name && args[0] != stack.ID {
					continue
				}

				printer.print(stack)
			}
		},
	}

	return infoCmd
}

type infoPrinter struct {
	w   *tabwriter.Writer
	aws *awsprov.AwsProvider
}

func newInfoPrinter(w io.Writer, aws *awsprov.AwsProvider) infoPrinter {
	return infoPrinter{
		w:   tabwriter.NewWriter(w, 0, 0, 1, ' ', 0),
		aws: aws,
	}
}

func (p infoPrinter) print(stack stackassembly.Stack) {
	p.printStackName(stack.Name)

	resources, err := p.aws.StackResources(stack.Name)
	handleError(err)

	p.printResources(resources)

	outputs, err := p.aws.StackOutputs(stack.Name)
	handleError(err)

	p.printOutputs(outputs)

	events, err := p.aws.StackEvents(stack.Name)
	handleError(err)

	p.printEvents(events)

	fmt.Println("")
}

func (p infoPrinter) printStackName(name string) {
	fmt.Println("===================")
	fmt.Printf("STACK: %s\n", name)
	fmt.Println("===================")
	fmt.Println("")
}

func (p infoPrinter) printResources(resources []stackassembly.StackResource) {
	fmt.Println("==== RESOURCES ====")
	for _, res := range resources {
		fmt.Fprintf(p.w, "%s\t%s\t%s\n", res.LogicalID, res.PhysicalID, res.Status)
	}
	p.w.Flush()
	fmt.Println("")
}

func (p infoPrinter) printOutputs(outputs []stackassembly.StackOutput) {
	fmt.Println("==== OUTPUTS ====")
	for _, out := range outputs {
		fmt.Fprintf(p.w, "%s\t%s\t%s\n", out.Key, out.Value, out.ExportName)
	}
	p.w.Flush()
	fmt.Println("")
}

func (p infoPrinter) printEvents(events []stackassembly.StackEvent) {
	fmt.Println("==== EVENTS ====")
	limit := 10
	if len(events) < limit {
		limit = len(events)
	}
	for _, e := range events[:limit] {
		fmt.Fprintf(p.w, "[%v]\t%s\t%s\t%s\t%s\n", e.Timestamp, e.ResourceType, e.LogicalResourceID, e.Status, e.StatusReason)
	}
	p.w.Flush()
	fmt.Println("")
}
