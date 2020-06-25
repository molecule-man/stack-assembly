package cli

import (
	"bytes"
	"fmt"
	"strings"
)

// Table represents data as a table.
type Table struct {
	buf        *bytes.Buffer
	currentRow int
	w          *ColWriter
}

// NewTable creates a new table.
func NewTable() *Table {
	buf := bytes.NewBuffer([]byte{})
	w := NewColWriter(buf, " | ")
	w.PadLastColumn = true
	w.DecorateLine = func(line string) string {
		if strings.HasPrefix(line, "-") {
			line = strings.Replace(line, "|", "+", -1)
			line = strings.Replace(line, " ", "-", -1)

			return "+-" + line + "-+"
		}

		return "| " + line + " |"
	}

	return &Table{buf: buf, w: w}
}

func (t *Table) Header(cells ...string) *Table {
	t.Row(cells...)
	fmt.Fprintln(t.w, "-")

	return t
}

func (t *Table) Row(cells ...string) *Table {
	if t.currentRow == 0 {
		fmt.Fprintln(t.w, "-")
	}

	fmt.Fprintln(t.w, strings.Join(cells, "\t"))
	t.currentRow++

	return t
}

// Render renders the table.
func (t *Table) Render() string {
	fmt.Fprintln(t.w, "-")
	t.w.Flush()

	return t.buf.String()
}
