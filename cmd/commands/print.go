package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/molecule-man/stack-assembly/cli"
	"github.com/molecule-man/stack-assembly/cli/color"
	"github.com/molecule-man/stack-assembly/stackassembly"
)

func colorizedStatus(status string) string {
	switch {
	case strings.Contains(status, "COMPLETE"):
		return color.Success(status)
	case strings.Contains(status, "ROLLBACK"), strings.Contains(status, "FAILED"):
		return color.Fail(status)
	}

	return status
}

func sprintEvent(e stackassembly.StackEvent) string {
	return fmt.Sprintf("%s\t%s\t%s\t%s", e.ResourceType, colorizedStatus(e.Status), e.LogicalResourceID, e.StatusReason)
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
	cli.Print(fmt.Sprintf("STATUS:\t%s %s", colorizedStatus(info.Status()), info.StatusDescription()))
	cli.Print("")
}

func printResources(stack *stackassembly.Stack) {
	resources, err := stack.Resources()
	handleError(err)

	cli.Print("==== RESOURCES ====")

	w := cli.NewColWriter(cli.Output, " ")
	for _, res := range resources {
		fields := []string{res.LogicalID, colorizedStatus(res.Status), res.PhysicalID}
		fmt.Fprintln(w, strings.Join(fields, "\t"))
	}
	handleError(w.Flush())
	cli.Print("")
}

func printOutputs(info stackassembly.StackInfo) {
	cli.Print("==== OUTPUTS ====")

	w := cli.NewColWriter(cli.Output, " ")
	for _, out := range info.Outputs() {
		fmt.Fprintln(w, strings.Join([]string{out.Key, out.Value, out.ExportName}, "\t"))
	}
	handleError(w.Flush())
	cli.Print("")
}

func printEvents(stack *stackassembly.Stack) {
	events, err := stack.Events()
	handleError(err)

	w := cli.NewColWriter(cli.Output, " ")
	cli.Print("==== EVENTS ====")
	limit := 10
	if len(events) < limit {
		limit = len(events)
	}
	for _, e := range events[:limit] {
		fmt.Fprintf(w, "[%s]\t%s\n", e.Timestamp.Format(time.RFC3339), sprintEvent(e))
	}
	handleError(w.Flush())
	cli.Print("")
}
