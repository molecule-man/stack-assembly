package commands

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/molecule-man/stack-assembly/awsprov"
	"github.com/molecule-man/stack-assembly/cli"
	"github.com/molecule-man/stack-assembly/stackassembly"
)

func colorForStatus(status string) *color.Color {
	col := noColor

	switch {
	case strings.Contains(status, "COMPLETE"):
		col = green
	case strings.Contains(status, "ROLLBACK"),
		strings.Contains(status, "FAILED"):
		col = boldRed
	}

	return col
}

func sprintStackStatus(status string) string {
	return colorForStatus(status).Sprint(status)
}

func sprintEvent(e stackassembly.StackEvent) string {
	return fmt.Sprintf("\t%s\t%s\t%s\t%s", e.ResourceType, sprintStackStatus(e.Status), e.LogicalResourceID, e.StatusReason)
}

type infoPrinter struct {
	aws *awsprov.AwsProvider
}

func newInfoPrinter(aws *awsprov.AwsProvider) infoPrinter {
	return infoPrinter{
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
	t := cli.NewTable()
	t.NoBorder()
	fmt.Println("==== RESOURCES ====")
	for _, res := range resources {
		t.Row()

		t.Cell(res.LogicalID)
		t.ColorizedCell(res.Status, colorForStatus(res.Status))
		t.Cell(res.PhysicalID)
	}
	fmt.Println(t.Render())
}

func (p infoPrinter) printOutputs(outputs []stackassembly.StackOutput) {
	t := cli.NewTable()
	t.NoBorder()
	fmt.Println("==== OUTPUTS ====")
	for _, out := range outputs {
		t.Row().Cell(out.Key).Cell(out.Value).Cell(out.ExportName)
	}
	fmt.Println(t.Render())
}

func (p infoPrinter) printEvents(events []stackassembly.StackEvent) {
	t := cli.NewTable()
	t.NoBorder()
	fmt.Println("==== EVENTS ====")
	limit := 10
	if len(events) < limit {
		limit = len(events)
	}
	for _, e := range events[:limit] {
		t.Row()

		t.Cell(fmt.Sprintf("[%v]", e.Timestamp))
		t.Cell(e.ResourceType)
		t.ColorizedCell(e.Status, colorForStatus(e.Status))
		t.Cell(e.LogicalResourceID)
		t.Cell(e.StatusReason)
	}
	fmt.Println(t.Render())
}
