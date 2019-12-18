package commands

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func (c *Commands) InvokeAwscliIfNeeded() {
	if !c.IsAwsDropIn() {
		return
	}

	subCmd, _, err := c.RootCmd().Find(os.Args[1:])

	if err == nil && subCmd.Runnable() {
		c.origArgs = os.Args
		os.Args = normalizeAwsParams([]string{
			"--parameter-overrides", "--capabilities", "--tags",
			"--notification-arns", "--parameters",
		}, os.Args)

		return
	}

	c.InvokeAwscli()
}

func (c *Commands) IsAwsDropIn() bool {
	return len(os.Args) > 0 && os.Args[0] == "aws"
}

func (c *Commands) InvokeAwscli() {
	args := c.origArgs
	if len(args) == 0 {
		args = os.Args
	}

	cmd := exec.Command("/usr/local/bin/aws", args[1:]...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
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

var awsParamsRe = regexp.MustCompile(`\w+=\w+(,\w+=\w+)*`)

func awsParamsToMap(params []string) (map[string]string, error) {
	mappedParams := make(map[string]string, len(params))

	for _, p := range params {
		if !awsParamsRe.MatchString(p) {
			return mappedParams, fmt.Errorf("not able to parse parameter %s: %w", p, ErrAwsDropIn)
		}

		parts := strings.Split(p, ",")
		key := ""
		val := ""

		for _, part := range parts {
			keyVal := strings.Split(part, "=")

			switch keyVal[0] {
			case "ParameterKey":
				key = keyVal[1]
			case "ParameterValue":
				val = keyVal[1]
			case "UsePreviousValue":
				// do nothing
			default:
				return mappedParams, fmt.Errorf("%s is not supported: %w", keyVal[0], ErrAwsDropIn)
			}
		}

		if key != "" && val != "" {
			mappedParams[key] = val
		}
	}

	return mappedParams, nil
}

var ErrAwsDropIn = errors.New("aws drop in error")

func IsAwsDropInError(err error) bool {
	return errors.Is(err, ErrAwsDropIn) || strings.HasPrefix(err.Error(), "unknown flag")
}
