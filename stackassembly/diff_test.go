package stackassembly

import (
	"errors"
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

	cf := &cfMock{}
	cf.body = oldTplBody
	st := Stack{
		Name: "teststack",
		body: newTplBody,
		cf:   cf,
	}

	diff, err := Diff(st)
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

	cf := &cfMock{}
	cf.describeErr = errors.New("stack does not exist")
	st := Stack{
		Name: "teststack",
		body: newTplBody,
		cf:   cf,
	}

	diff, err := Diff(st)
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
