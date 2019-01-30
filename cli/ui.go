package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

var ErrPromptCommandIsNotKnown = errors.New("prompt command is not known")

type PromptCmd struct {
	TriggerInputs []string
	Description   string
	Action        func()
}

func Prompt(commands []PromptCmd) error {
	Print("*** Commands ***")
	for _, c := range commands {
		fmt.Println("  " + c.Description)
	}
	response, err := Ask("What now> ")
	if err != nil {
		return err
	}

	for _, c := range commands {
		for _, inp := range c.TriggerInputs {
			if inp == response {
				c.Action()
				return nil
			}
		}
	}

	Warnf("Command %s is not known\n", response)
	return nil
}

func Ask(query string, args ...interface{}) (string, error) {
	Printf(query, args...)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')

	if err != nil {
		return "", fmt.Errorf("reading user input failed with err: %v", err)
	}

	return strings.TrimSpace(response), nil
}
