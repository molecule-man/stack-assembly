package stackassembly

import (
	"strings"

	"github.com/pmezard/go-difflib/difflib"
)

type detailsProvider interface {
	StackDetails(stackName string) (StackDetails, error)
	StackExists(stackName string) (bool, error)
}

type DiffService struct {
	Dp detailsProvider
}

func (ds DiffService) Diff(stack Stack) (string, error) {
	// TODO diff not only body but also parameters and tags
	oldBody := ""
	oldName := "/dev/null"

	exists, err := ds.Dp.StackExists(stack.Name)
	if err != nil {
		return "", err
	}

	if exists {
		details, derr := ds.Dp.StackDetails(stack.Name)
		if derr != nil {
			return "", derr
		}

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
