package awscf

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/molecule-man/stack-assembly/cli"
	"github.com/pmezard/go-difflib/difflib"
	"gopkg.in/yaml.v3"
)

const defaultDiffName = "/dev/null"

type ChSetDiff struct {
	Color cli.Color
}

func (d ChSetDiff) Diff(chSet *ChangeSet) (string, error) {
	diffs := []string{}

	paramsDiff, err := diffParameters(chSet)
	if err != nil {
		return "", err
	}

	if len(paramsDiff) > 0 {
		diffs = append(diffs, d.colorizeDiff(paramsDiff))
	}

	tagsDiff, err := diffTags(chSet)
	if err != nil {
		return "", err
	}

	if len(tagsDiff) > 0 {
		diffs = append(diffs, d.colorizeDiff(tagsDiff))
	}

	bodyDiff, err := diffBody(chSet)
	if err != nil {
		return "", err
	}

	if len(bodyDiff) > 0 {
		diffs = append(diffs, d.colorizeDiff(bodyDiff))
	}

	return strings.Join(diffs, "\n"), nil
}

func diffBody(chSet *ChangeSet) (string, error) {
	if chSet.body == "" {
		return "", nil
	}

	oldBody := ""
	oldName := defaultDiffName

	deployed, err := chSet.Stack().AlreadyDeployed()
	if err != nil {
		return "", err
	}

	if deployed {
		oldBody, err = chSet.Stack().Body()
		if err != nil {
			return "", err
		}

		oldName = "old/" + chSet.Stack().Name
	}

	var (
		oldBodyRaw interface{}
		bodyRaw    interface{}
	)

	newBodyBytes := []byte(chSet.body)
	oldBodyBytes := []byte(oldBody)
	oldBodyYAML := false
	oldBodyJSON := false
	newBodyYAML := false

	if oldBody != "" {
		jsonErr := json.Unmarshal(oldBodyBytes, &oldBodyRaw)
		oldBodyJSON = jsonErr == nil

		if !oldBodyJSON {
			yamlErr := yaml.Unmarshal(oldBodyBytes, &oldBodyRaw)
			oldBodyYAML = yamlErr == nil
		}
	}

	jsonErr := json.Unmarshal(newBodyBytes, &bodyRaw)
	newBodyJSON := jsonErr == nil

	if !newBodyJSON {
		yamlErr := yaml.Unmarshal(newBodyBytes, &bodyRaw)
		newBodyYAML = yamlErr == nil
	}

	if newBodyYAML && oldBodyYAML && reflect.DeepEqual(bodyRaw, oldBodyRaw) {
		return "", nil
	}

	if newBodyJSON || oldBodyJSON {
		newBodyBytes, err = json.MarshalIndent(bodyRaw, "", "  ")
		if err != nil {
			return "", err
		}

		oldBodyBytes, err = json.MarshalIndent(oldBodyRaw, "", "  ")
		if err != nil {
			return "", err
		}
	}

	return difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(strings.TrimSpace(string(oldBodyBytes))),
		B:        difflib.SplitLines(strings.TrimSpace(string(newBodyBytes))),
		FromFile: oldName,
		FromDate: "",
		ToFile:   "new/" + chSet.Stack().Name,
		ToDate:   "",
		Context:  5,
	})
}

func diffParameters(chSet *ChangeSet) (string, error) {
	awsParams, err := chSet.awsParameters()
	if err != nil {
		return "", err
	}

	newParams := make([]string, 0, len(awsParams))

	for _, p := range awsParams {
		line := aws.StringValue(p.ParameterKey) + ": " + aws.StringValue(p.ParameterValue) + "\n"
		newParams = append(newParams, line)
	}

	oldName := defaultDiffName
	oldParams := []string{}

	deployed, err := chSet.Stack().AlreadyDeployed()
	if err != nil {
		return "", err
	}

	if deployed {
		info, err := chSet.Stack().Info()
		if err != nil {
			return "", err
		}

		oldName = "old-parameters/" + chSet.Stack().Name
		oldParams = make([]string, 0, len(info.Parameters()))

		for _, p := range info.Parameters() {
			oldParams = append(oldParams, p.Key+": "+p.Val+"\n")
		}
	}

	return difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        oldParams,
		B:        newParams,
		FromFile: oldName,
		FromDate: "",
		ToFile:   "new-parameters/" + chSet.Stack().Name,
		ToDate:   "",
		Context:  5,
	})
}

func diffTags(chSet *ChangeSet) (string, error) {
	newTags := make([]string, 0, len(chSet.tags))

	for k, v := range chSet.tags {
		newTags = append(newTags, k+": "+v+"\n")
	}

	oldName := defaultDiffName
	oldTags := []string{}

	deployed, err := chSet.Stack().AlreadyDeployed()
	if err != nil {
		return "", err
	}

	if deployed {
		info, err := chSet.Stack().Info()
		if err != nil {
			return "", err
		}

		oldName = "old-tags/" + chSet.Stack().Name
		oldTags = make([]string, 0, len(info.Tags()))

		for _, t := range info.Tags() {
			oldTags = append(oldTags, t.Key+": "+t.Val+"\n")
		}
	}

	sort.Strings(oldTags)
	sort.Strings(newTags)

	return difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        oldTags,
		B:        newTags,
		FromFile: oldName,
		FromDate: "",
		ToFile:   "new-tags/" + chSet.Stack().Name,
		ToDate:   "",
		Context:  5,
	})
}

func (d ChSetDiff) colorizeDiff(diff string) string {
	if d.Color.Disabled {
		return diff
	}

	colorized := strings.Split(diff, "\n")

	for i, line := range colorized {
		switch {
		case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"):
			colorized[i] = d.Color.Yellow(line)
		case strings.HasPrefix(line, "@@"):
			colorized[i] = d.Color.Cyan(line)
		case strings.HasPrefix(line, "+"):
			colorized[i] = d.Color.Green(line)
		case strings.HasPrefix(line, "-"):
			colorized[i] = d.Color.Red(line)
		}
	}

	return strings.Join(colorized, "\n")
}
