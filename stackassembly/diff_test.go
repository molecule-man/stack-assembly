package stackassembly

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiffWhenStackExists(t *testing.T) {
	oldTplBody := `
parameters:
  param1: old_val1
  param2: old_val2`
	newTplBody := `
parameters:
  param1: new_val1
  param2: old_val2`

	ds := DiffService{
		Dp: &dpMock{exists: true, body: oldTplBody},
	}
	st := StackConfig{
		Name: "teststack",
		Body: newTplBody,
	}

	diff, err := ds.Diff(st)
	require.NoError(t, err)

	expected := `
--- old/teststack
+++ new/teststack
@@ -1,3 +1,3 @@
 parameters:
-  param1: old_val1
+  param1: new_val1
   param2: old_val2
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(diff))
}

func TestDiffWhenStackDoesntExist(t *testing.T) {
	newTplBody := `
parameters:
  param1: val1
  param2: val2`

	ds := DiffService{Dp: &dpMock{}}
	st := StackConfig{
		Name: "teststack",
		Body: newTplBody,
	}

	diff, err := ds.Diff(st)
	require.NoError(t, err)

	expected := `
--- /dev/null
+++ new/teststack
@@ -1 +1,3 @@
-
+parameters:
+  param1: val1
+  param2: val2
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(diff))
}

type dpMock struct {
	exists bool
	body   string
	err    error
}

func (dpm *dpMock) StackDetails(name string) (StackDetails, error) {
	return StackDetails{Body: dpm.body}, dpm.err
}

func (dpm *dpMock) StackExists(name string) (bool, error) {
	return dpm.exists, dpm.err
}
