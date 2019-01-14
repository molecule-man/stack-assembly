package commands

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/molecule-man/stack-assembly/cli"
	"github.com/molecule-man/stack-assembly/stackassembly"
)

func colorForStatus(status string) *color.Color {
	switch {
	case strings.Contains(status, "COMPLETE"):
		return cli.SuccessColor
	case strings.Contains(status, "ROLLBACK"),
		strings.Contains(status, "FAILED"):
		return cli.FailureColor
	}

	return cli.NoColor
}

func sprintStackStatus(status string) string {
	return colorForStatus(status).Sprint(status)
}

func sprintEvent(e stackassembly.StackEvent) string {
	return fmt.Sprintf("\t%s\t%s\t%s\t%s", e.ResourceType, sprintStackStatus(e.Status), e.LogicalResourceID, e.StatusReason)
}

func printStackInfo(stack *stackassembly.Stack) {
	exists, err := stack.Exists()
	handleError(err)

	if !exists {
		return
	}

	info, err := stack.Info()
	handleError(err)

	printStackDetails(stack.Name, info)
	printResources(stack)
	printOutputs(info)
	printEvents(stack)

	cli.Print("")
}

func printStackDetails(name string, info stackassembly.StackInfo) {
	cli.Print("######################################")
	cli.Print(fmt.Sprintf("STACK:\t%s", name))
	cli.Print(fmt.Sprintf("STATUS:\t%s %s", sprintStackStatus(info.Status()), info.StatusDescription()))
	cli.Print("")
}

func printResources(stack *stackassembly.Stack) {
	resources, err := stack.Resources()
	handleError(err)

	t := cli.NewTable()
	t.NoBorder()
	cli.Print("==== RESOURCES ====")
	for _, res := range resources {
		t.Row()

		t.Cell(res.LogicalID)
		t.ColorizedCell(res.Status, colorForStatus(res.Status))
		t.Cell(res.PhysicalID)
	}
	cli.Print(t.Render())
}

func printOutputs(info stackassembly.StackInfo) {
	t := cli.NewTable()
	t.NoBorder()
	cli.Print("==== OUTPUTS ====")
	for _, out := range info.Outputs() {
		t.Row().Cell(out.Key).Cell(out.Value).Cell(out.ExportName)
	}
	cli.Print(t.Render())
}

func printEvents(stack *stackassembly.Stack) {
	events, err := stack.Events()
	handleError(err)

	t := cli.NewTable()
	t.NoBorder()
	cli.Print("==== EVENTS ====")
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
	cli.Print(t.Render())
}
