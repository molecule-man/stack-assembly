package cli

import (
	"fmt"
	"strconv"
	"strings"
)

type Table struct {
	currentRow int
	colSizes   []int
	data       [][]string
	hasHeader  bool
}

func NewTable() *Table {
	return &Table{
		currentRow: -1,
	}
}

func (t *Table) Header() *Table {
	t.hasHeader = true
	return t.Row()
}

func (t *Table) Row() *Table {
	t.currentRow++
	t.data = append(t.data, make([]string, 0))
	return t
}

func (t *Table) Cell(s string) *Table {
	currentCell := len(t.data[t.currentRow])

	if currentCell >= len(t.colSizes) {
		t.colSizes = append(t.colSizes, 0)
	}

	if len(s) > t.colSizes[currentCell] {
		t.colSizes[currentCell] = len(s)
	}

	t.data[t.currentRow] = append(t.data[t.currentRow], s)

	return t
}

func (t *Table) Render() string {
	renderedRows := make([]string, 0)
	borderParts := make([]string, len(t.colSizes))

	for i, size := range t.colSizes {
		borderParts[i] = strings.Repeat("-", size)
	}
	border := "+-" + strings.Join(borderParts, "-+-") + "-+"

	renderedRows = append(renderedRows, border)

	for r, row := range t.data {
		renderedCells := make([]string, len(t.colSizes))

		for c, size := range t.colSizes {
			cell := ""

			if c < len(row) {
				cell = row[c]
			}

			renderedCells[c] = fmt.Sprintf("%-"+strconv.Itoa(size)+"s", cell)
		}
		renderedRows = append(renderedRows, "| "+strings.Join(renderedCells, " | ")+" |")

		if r == 0 && t.hasHeader {
			renderedRows = append(renderedRows, border)
		}
	}

	renderedRows = append(renderedRows, border)

	return strings.Join(renderedRows, "\n") + "\n"
}
