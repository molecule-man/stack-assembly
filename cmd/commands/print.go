package commands

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
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

func printStackInfo(stack stackassembly.Stack) {
	printStackDetails(&stack)
	printResources(&stack)
	printOutputs(&stack)
	printEvents(&stack)

	fmt.Println("")
}

func printStackDetails(stack *stackassembly.Stack) {
	status, err := stack.Status()
	handleError(err)
	fmt.Println("######################################")
	fmt.Printf("STACK:\t%s\n", stack.Name)
	fmt.Printf("STATUS:\t%s %s\n", sprintStackStatus(status.Status), status.StatusDescription)
	fmt.Println("")
}

func printResources(stack *stackassembly.Stack) {
	resources, err := stack.Resources()
	handleError(err)

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

func printOutputs(stack *stackassembly.Stack) {
	outputs, err := stack.Outputs()
	handleError(err)

	t := cli.NewTable()
	t.NoBorder()
	fmt.Println("==== OUTPUTS ====")
	for _, out := range outputs {
		t.Row().Cell(out.Key).Cell(out.Value).Cell(out.ExportName)
	}
	fmt.Println(t.Render())
}

func printEvents(stack *stackassembly.Stack) {
	events, err := stack.Events()
	handleError(err)

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
