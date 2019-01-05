package stackassembly

import (
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/fatih/color"
	"github.com/pmezard/go-difflib/difflib"
)

const defaultDiffName = "/dev/null"

func Diff(stack Stack) (string, error) {
	info, err := stack.Info()
	if err != nil && err != ErrStackDoesntExist {
		return "", err
	}

	diffs := []string{}

	paramsDiff, err := diffParameters(info, stack)
	if err != nil {
		return "", err
	}
	if len(paramsDiff) > 0 {
		diffs = append(diffs, colorizeDiff(paramsDiff))
	}

	tagsDiff, err := diffTags(info, stack)
	if err != nil {
		return "", err
	}
	if len(tagsDiff) > 0 {
		diffs = append(diffs, colorizeDiff(tagsDiff))
	}

	bodyDiff, err := diffBody(info.Exists(), stack)
	if err != nil {
		return "", err
	}
	if len(bodyDiff) > 0 {
		diffs = append(diffs, colorizeDiff(bodyDiff))
	}

	return strings.Join(diffs, "\n"), nil
}

func diffBody(stackExists bool, stack Stack) (string, error) {
	oldBody := ""
	oldName := defaultDiffName

	var err error

	if stackExists {
		oldBody, err = stack.RemoteBody()
		if err != nil {
			return "", err
		}
		oldName = "old/" + stack.Name
	}

	newBody, err := stack.Body()
	if err != nil {
		return "", err
	}

	return difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(strings.TrimSpace(oldBody)),
		B:        difflib.SplitLines(strings.TrimSpace(newBody)),
		FromFile: oldName,
		FromDate: "",
		ToFile:   "new/" + stack.Name,
		ToDate:   "",
		Context:  5,
	})
}

func colorizeDiff(diff string) string {
	if color.NoColor {
		return diff
	}

	colorized := strings.Split(diff, "\n")

	for i, line := range colorized {
		switch {
		case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"):
			colorized[i] = color.YellowString(line)
		case strings.HasPrefix(line, "@@"):
			colorized[i] = color.CyanString(line)
		case strings.HasPrefix(line, "+"):
			colorized[i] = color.GreenString(line)
		case strings.HasPrefix(line, "-"):
			colorized[i] = color.RedString(line)
		}

	}

	return strings.Join(colorized, "\n")
}

func diffParameters(info StackInfo, stack Stack) (string, error) {
	awsParams, err := stack.awsParameters()
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

	if info.Exists() {
		parameters := info.Parameters()
		oldName = "old-parameters/" + stack.Name
		oldParams = make([]string, 0, len(parameters))

		for _, p := range parameters {
			oldParams = append(oldParams, p.Key+": "+p.Val+"\n")
		}
	}

	return difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        oldParams,
		B:        newParams,
		FromFile: oldName,
		FromDate: "",
		ToFile:   "new-parameters/" + stack.Name,
		ToDate:   "",
		Context:  5,
	})
}

func diffTags(info StackInfo, stack Stack) (string, error) {
	newTags := make([]string, 0, len(stack.tags))

	for k, v := range stack.tags {
		newTags = append(newTags, k+": "+v+"\n")
	}

	oldName := defaultDiffName
	oldTags := []string{}

	if info.Exists() {
		oldName = "old-tags/" + stack.Name
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
		ToFile:   "new-tags/" + stack.Name,
		ToDate:   "",
		Context:  5,
	})
}
