package commands

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"

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

			aws := conf.Aws(cfg)

			printer := newInfoPrinter(os.Stdout)

			for k, stackCfg := range cfg.Stacks {

				if len(args) > 0 && args[0] != stackCfg.Name && args[0] != k {
					continue
				}

				stack, err := stackassembly.NewStack("", stackCfg, cfg.Parameters)
				handleError(err)

				printer.printStackName(stack.Name)

				resources, err := aws.StackResources(stack.Name)
				handleError(err)

				printer.printResources(resources)

				outputs, err := aws.StackOutputs(stack.Name)
				handleError(err)

				printer.printOutputs(outputs)

				events, err := aws.StackEvents(stack.Name)
				handleError(err)

				printer.printEvents(events)

				fmt.Println("")
			}
		},
	}

	return infoCmd
}

type infoPrinter struct {
	w *tabwriter.Writer
}

func newInfoPrinter(w io.Writer) infoPrinter {
	return infoPrinter{
		w: tabwriter.NewWriter(w, 0, 0, 1, ' ', 0),
	}
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
	for _, e := range events[:10] {
		fmt.Fprintf(p.w, "[%v]\t%s\t%s\t%s\t%s\n", e.Timestamp, e.ResourceType, e.LogicalResourceID, e.Status, e.StatusReason)
	}
	p.w.Flush()
	fmt.Println("")
}
