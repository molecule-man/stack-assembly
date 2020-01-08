package commands

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func (c *Commands) NormalizeAwscliParamsIfNeeded() {
	c.origArgs = os.Args
	os.Args = normalizeAwsParams([]string{
		"--parameter-overrides", "--capabilities", "--tags",
		"--notification-arns", "--parameters",
	}, os.Args)
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
