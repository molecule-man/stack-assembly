package cli

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestColWriter(t *testing.T) {
	buf := bytes.NewBuffer([]byte{})
	writer := NewColWriter(buf, " ")
	writer.PadLastColumn = true

	fmt.Fprintln(writer, "hello\tWorld")
	fmt.Fprintln(writer, "a\tlong long long long...")
	fmt.Fprintln(writer, "b")

	require.NoError(t, writer.Flush())

	expected := strings.Join([]string{
		"hello World                 ",
		"a     long long long long...",
		"b                           \n",
	}, "\n")
	assert.Equal(t, expected, buf.String())
}

func TestColWriterWithColors(t *testing.T) {
	buf := bytes.NewBuffer([]byte{})
	writer := NewColWriter(buf, " | ")

	color := Color{}

	fmt.Fprintln(writer, color.Fail("red")+"\t"+color.Success("green"))
	fmt.Fprintln(writer, color.Neutral("cyan")+"\t"+color.Warn("multi")+" "+color.Fail("color"))

	require.NoError(t, writer.Flush())

	expected := strings.Join([]string{
		"\x1b[31;1mred\x1b[0m  | \x1b[32mgreen\x1b[0m",
		"\x1b[36mcyan\x1b[0m | \x1b[33mmulti\x1b[0m \x1b[31;1mcolor\x1b[0m\n",
	}, "\n")
	assert.Equal(t, expected, buf.String())
}

func TestContinuousFlushing(t *testing.T) {
	buf := bytes.NewBuffer([]byte{})
	writer := NewColWriter(buf, " ")

	fmt.Fprintln(writer, "resource_x\tUPDATE_COMPLETE\tresource_x updated")
	writer.Flush()
	fmt.Fprintln(writer, "resource_xy\tUPDATE_IN_PROGRESS\tresource_xy is being updated")
	writer.Flush()
	fmt.Fprintln(writer, "resource_x\tCREATE_COMPLETE\tresource_x created")
	writer.Flush()

	require.NoError(t, writer.Flush())

	expected := strings.Join([]string{
		"resource_x UPDATE_COMPLETE resource_x updated",
		"resource_xy UPDATE_IN_PROGRESS resource_xy is being updated",
		"resource_x  CREATE_COMPLETE    resource_x created\n",
	}, "\n")
	assert.Equal(t, expected, buf.String())
}
