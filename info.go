package assembly

import (
	"fmt"
	"strings"
	"time"

	"github.com/molecule-man/stack-assembly/awscf"
	"github.com/molecule-man/stack-assembly/cli"
	"github.com/molecule-man/stack-assembly/conf"
)

func (sa SA) colorizedStatus(status string) string {
	switch {
	case strings.HasSuffix(status, "COMPLETE"):
		return sa.cli.Color.Success(status)
	case strings.HasSuffix(status, "IN_PROGRESS"):
		return sa.cli.Color.Neutral(status)
	case strings.Contains(status, "ROLLBACK"), strings.Contains(status, "FAILED"):
		return sa.cli.Color.Fail(status)
	}

	return status
}

func (sa SA) sprintEvent(e awscf.StackEvent) string {
	return fmt.Sprintf("%s\t%s\t%s\t%s", e.ResourceType, sa.colorizedStatus(e.Status), e.LogicalResourceID, e.StatusReason)
}

func (sa SA) Info(stack *awscf.Stack) error {
	exists, err := stack.Exists()
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	info, err := stack.Info()
	if err != nil {
		return err
	}

	sa.printStackDetails(stack.Name, info)
	sa.printResources(stack)
	sa.printOutputs(info)
	sa.printEvents(stack)

	sa.cli.Print("")

	return nil
}

func (sa SA) InfoAll(cfg conf.Config) error {
	ss, err := cfg.StackConfigsSortedByExecOrder()
	if err != nil {
		return err
	}

	for _, s := range ss {
		err = sa.InfoAll(s)
		if err != nil {
			return err
		}
	}

	if cfg.Name == "" {
		return nil
	}

	return sa.Info(cfg.Stack())
}

func (sa SA) printStackDetails(name string, info awscf.StackInfo) {
	sa.cli.Print("######################################")
	sa.cli.Print(fmt.Sprintf("STACK:\t%s", name))
	sa.cli.Print(fmt.Sprintf("STATUS:\t%s %s", sa.colorizedStatus(info.Status()), info.StatusDescription()))
	sa.cli.Print("")
}

func (sa SA) printResources(stack *awscf.Stack) {
	resources, err := stack.Resources()
	MustSucceed(err)

	sa.cli.Print("==== RESOURCES ====")

	w := cli.NewColWriter(sa.cli.Writer, " ")

	for _, res := range resources {
		fields := []string{res.LogicalID, sa.colorizedStatus(res.Status), res.PhysicalID}
		fmt.Fprintln(w, strings.Join(fields, "\t"))
	}

	MustSucceed(w.Flush())
	sa.cli.Print("")
}

func (sa SA) printOutputs(info awscf.StackInfo) {
	sa.cli.Print("==== OUTPUTS ====")

	w := cli.NewColWriter(sa.cli.Writer, " ")

	for _, out := range info.Outputs() {
		fmt.Fprintln(w, strings.Join([]string{out.Key, out.Value, out.ExportName}, "\t"))
	}

	MustSucceed(w.Flush())
	sa.cli.Print("")
}

func (sa SA) printEvents(stack *awscf.Stack) {
	events, err := stack.Events()
	MustSucceed(err)

	w := cli.NewColWriter(sa.cli.Writer, " ")
	sa.cli.Print("==== EVENTS ====")

	limit := 10
	if len(events) < limit {
		limit = len(events)
	}

	for _, e := range events[:limit] {
		fmt.Fprintf(w, "[%s]\t%s\n", e.Timestamp.Format(time.RFC3339), sa.sprintEvent(e))
	}

	MustSucceed(w.Flush())
	sa.cli.Print("")
}
