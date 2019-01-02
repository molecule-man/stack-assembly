package commands

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/molecule-man/stack-assembly/awsprov"
	"github.com/molecule-man/stack-assembly/stackassembly"
)

func sprintEvent(e stackassembly.StackEvent) string {
	col := noColor

	switch {
	case strings.Contains(e.Status, "COMPLETE"):
		col = green
	case strings.Contains(e.Status, "ROLLBACK"),
		strings.Contains(e.Status, "FAILED"):
		col = boldRed
	}
	return fmt.Sprintf("\t%s\t%s\t%s\t%s", e.ResourceType, col.Sprint(e.Status), e.LogicalResourceID, e.StatusReason)
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
		fmt.Fprintf(p.w, "[%v]%s\n", e.Timestamp, sprintEvent(e))
	}
	p.w.Flush()
	fmt.Println("")
}
