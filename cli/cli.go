// Package cli provides cli things
package cli

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/molecule-man/claws/claws"
)

// Approval enables user confirmation to apply stack changes
type Approval struct {
}

// Approve asks user for confirmation
func (a *Approval) Approve(changes []claws.Change) bool {
	showChanges(changes)
	return askConfirmation()
}

func askConfirmation() bool {
	fmt.Print("Continue? [Y/n] ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')

	if err != nil {
		log.Fatalf("Reading user input failed with err: %v", err)
	}

	response = strings.Trim(response, " \n")

	for _, okayResponse := range []string{"", "y", "Y", "yes", "Yes", "YES"} {
		if response == okayResponse {
			return true
		}
	}

	fmt.Println("Interrupted by user")
	return false
}

func showChanges(changes []claws.Change) {
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
