// Package cli provides cli things
package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/molecule-man/claws/claws"
)

type Approval struct {
}

func (a *Approval) Approve() bool {
	fmt.Print("Continue? [Y/n] ")

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')

	response = strings.Trim(response, " \n")

	for _, okayResponse := range []string{"", "y", "Y", "yes", "Yes", "YES"} {
		if response == okayResponse {
			return true
		}
	}

	fmt.Println("Interrupted by user")
	return false
}

type ChangeTable struct {
}

func (cp *ChangeTable) ShowChanges(changes []claws.Change) {
	if len(changes) > 0 {
		t := NewTable()
		t.Header().Cell("Action").Cell("ResourceType").Cell("Resource ID").Cell("Replacement needed")

		for _, c := range changes {
			t.Row().
				Cell(c.Action).
				Cell(c.ResourceType).
				Cell(c.LogicalResourceID).
				Cell(fmt.Sprintf("%t", c.ReplacementNeeded))
		}

		fmt.Println(t.Render())
	}
}
