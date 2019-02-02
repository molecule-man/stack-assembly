package assembly

import (
	"fmt"
	"strings"
	"time"

	"github.com/molecule-man/stack-assembly/awscf"
	"github.com/molecule-man/stack-assembly/cli"
	"github.com/molecule-man/stack-assembly/cli/color"
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

func sprintEvent(e awscf.StackEvent) string {
	return fmt.Sprintf("%s\t%s\t%s\t%s", e.ResourceType, colorizedStatus(e.Status), e.LogicalResourceID, e.StatusReason)
}

func Info(stack *awscf.Stack) {
	exists, err := stack.Exists()
	MustSucceed(err)

	if !exists {
		return
	}

	info, err := stack.Info()
	MustSucceed(err)

	printStackDetails(stack.Name, info)
	printResources(stack)
	printOutputs(info)
	printEvents(stack)

	cli.Print("")
}

func printStackDetails(name string, info awscf.StackInfo) {
	cli.Print("######################################")
	cli.Print(fmt.Sprintf("STACK:\t%s", name))
	cli.Print(fmt.Sprintf("STATUS:\t%s %s", colorizedStatus(info.Status()), info.StatusDescription()))
	cli.Print("")
}

func printResources(stack *awscf.Stack) {
	resources, err := stack.Resources()
	MustSucceed(err)

	cli.Print("==== RESOURCES ====")

	w := cli.NewColWriter(cli.Output, " ")
	for _, res := range resources {
		fields := []string{res.LogicalID, colorizedStatus(res.Status), res.PhysicalID}
		fmt.Fprintln(w, strings.Join(fields, "\t"))
	}
	MustSucceed(w.Flush())
	cli.Print("")
}

func printOutputs(info awscf.StackInfo) {
	cli.Print("==== OUTPUTS ====")

	w := cli.NewColWriter(cli.Output, " ")
	for _, out := range info.Outputs() {
		fmt.Fprintln(w, strings.Join([]string{out.Key, out.Value, out.ExportName}, "\t"))
	}
	MustSucceed(w.Flush())
	cli.Print("")
}

func printEvents(stack *awscf.Stack) {
	events, err := stack.Events()
	MustSucceed(err)

	w := cli.NewColWriter(cli.Output, " ")
	cli.Print("==== EVENTS ====")
	limit := 10
	if len(events) < limit {
		limit = len(events)
	}
	for _, e := range events[:limit] {
		fmt.Fprintf(w, "[%s]\t%s\n", e.Timestamp.Format(time.RFC3339), sprintEvent(e))
	}
	MustSucceed(w.Flush())
	cli.Print("")
}
