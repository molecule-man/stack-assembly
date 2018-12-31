package stackassembly

import (
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/pmezard/go-difflib/difflib"
)

const defaultDiffName = "/dev/null"

type detailsProvider interface {
	StackDetails(stackName string) (StackDetails, error)
	StackExists(stackName string) (bool, error)
	ValidateTemplate(tplBody string) ([]string, error)
}

type DiffService struct {
	Dp detailsProvider
}

func (ds DiffService) Diff(stack Stack) (string, error) {
	exists, err := ds.Dp.StackExists(stack.Name)
	if err != nil {
		return "", err
	}

	var details *StackDetails
	if exists {
		d, derr := ds.Dp.StackDetails(stack.Name)
		if derr != nil {
			return "", derr
		}
		details = &d
	}

	diffs := []string{}

	paramsDiff, err := ds.diffParameters(details, stack)
	if err != nil {
		return "", err
	}
	if len(paramsDiff) > 0 {
		diffs = append(diffs, colorizeDiff(paramsDiff))
	}

	tagsDiff, err := ds.diffTags(details, stack)
	if err != nil {
		return "", err
	}
	if len(tagsDiff) > 0 {
		diffs = append(diffs, colorizeDiff(tagsDiff))
	}

	bodyDiff, err := ds.diffBody(details, stack)
	if err != nil {
		return "", err
	}
	if len(bodyDiff) > 0 {
		diffs = append(diffs, colorizeDiff(bodyDiff))
	}

	return strings.Join(diffs, "\n"), nil
}

func (ds DiffService) diffBody(details *StackDetails, stack Stack) (string, error) {
	oldBody := ""
	oldName := defaultDiffName

	if details != nil {
		oldBody = details.Body
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

func (ds DiffService) diffParameters(details *StackDetails, stack Stack) (string, error) {
	if (details == nil || len(details.Parameters) == 0) && len(stack.Parameters) == 0 {
		return "", nil
	}

	body, err := stack.Body()
	if err != nil {
		return "", err
	}

	// TODO it looks strange to use ValidateTemplate only to get parameters
	requiredParams, err := ds.Dp.ValidateTemplate(body)
	if err != nil {
		return "", err
	}

	newParams := make([]string, 0, len(requiredParams))

	for _, p := range requiredParams {
		if v, ok := stack.Parameters[p]; ok {
			newParams = append(newParams, p+": "+v+"\n")
		}
	}

	oldName := defaultDiffName
	oldParams := []string{}

	if details != nil {
		oldName = "old-parameters/" + stack.Name
		oldParams = make([]string, 0, len(details.Parameters))

		for _, p := range details.Parameters {
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

func (ds DiffService) diffTags(details *StackDetails, stack Stack) (string, error) {
	if (details == nil || len(details.Tags) == 0) && len(stack.Tags) == 0 {
		return "", nil
	}

	newTags := make([]string, 0, len(stack.Tags))

	for k, v := range stack.Tags {
		newTags = append(newTags, k+": "+v+"\n")
	}

	oldName := defaultDiffName
	oldTags := []string{}

	if details != nil {
		oldName = "old-tags/" + stack.Name
		oldTags = make([]string, 0, len(details.Tags))

		for _, t := range details.Tags {
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
