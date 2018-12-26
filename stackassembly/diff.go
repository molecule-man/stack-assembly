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

func (ds DiffService) Diff(tpl StackConfig) (string, error) {
	oldBody := ""
	oldName := "/dev/null"

	exists, err := ds.Dp.StackExists(tpl.Name)
	if err != nil {
		return "", err
	}

	if exists {
		details, err := ds.Dp.StackDetails(tpl.Name)
		if err != nil {
			return "", err
		}

		oldBody = details.Body
		oldName = "old/" + tpl.Name
	}

	return difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(strings.TrimSpace(oldBody)),
		B:        difflib.SplitLines(strings.TrimSpace(tpl.Body)),
		FromFile: oldName,
		FromDate: "",
		ToFile:   "new/" + tpl.Name,
		ToDate:   "",
		Context:  5,
	})
}
