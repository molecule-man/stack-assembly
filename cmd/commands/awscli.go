package commands

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"strings"
)

func (c *Commands) InvokeAwscliIfNeeded() {
	if len(os.Args) == 0 || os.Args[0] != "aws" {
		return
	}

	subCmd, _, err := c.RootCmd().Find(os.Args[1:])

	if err == nil && subCmd.Runnable() {
		os.Args = normalizeAwsParams([]string{
			"--parameter-overrides", "--capabilities", "--tags",
			"--notification-arns",
		}, os.Args)

		return
	}

	cmd := exec.Command("/usr/local/bin/aws", os.Args[1:]...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err == nil {
		os.Exit(0)
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		os.Exit(exitErr.ExitCode())
	}

	log.Fatalf("aws cli exited unexpectedly: %v", err)
}

func normalizeAwsParams(flags []string, pp []string) []string {
	flagMap := make(map[string]bool, len(flags))
	for _, f := range flags {
		flagMap[f] = true
	}

	out := make([]string, 0, len(pp))

	for _, p := range pp {
		if len(out) > 1 &&
			!strings.HasPrefix(p, "--") &&
			flagMap[out[len(out)-2]] &&
			!strings.HasPrefix(out[len(out)-1], "--") {
			out = append(out, out[len(out)-2])
		}

		out = append(out, p)
	}

	return out
}
