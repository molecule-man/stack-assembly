package commands

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/molecule-man/stack-assembly/awsprov"
	"github.com/molecule-man/stack-assembly/stackassembly"
)

func sprintStackStatus(status string) string {
	col := noColor

	switch {
	case strings.Contains(status, "COMPLETE"):
		col = green
	case strings.Contains(status, "ROLLBACK"),
		strings.Contains(status, "FAILED"):
		col = boldRed
	}

	return col.Sprint(status)
}
func sprintEvent(e stackassembly.StackEvent) string {
	return fmt.Sprintf("\t%s\t%s\t%s\t%s", e.ResourceType, sprintStackStatus(e.Status), e.LogicalResourceID, e.StatusReason)
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
	details, err := p.aws.StackDetails(stack.Name)
	handleError(err)

	p.printStackDetails(details)

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

func (p infoPrinter) printStackDetails(details stackassembly.StackDetails) {
	fmt.Println("######################################")
	fmt.Printf("STACK:\t%s\n", details.Name)
	fmt.Printf("STATUS:\t%s %s\n", sprintStackStatus(details.Status), details.StatusDescription)
	fmt.Println("")
}

func (p infoPrinter) printResources(resources []stackassembly.StackResource) {
	fmt.Println("==== RESOURCES ====")
	for _, res := range resources {
		fmt.Fprintf(p.w, "%s\t%s\t%s\n", res.LogicalID, res.PhysicalID, sprintStackStatus(res.Status))
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
