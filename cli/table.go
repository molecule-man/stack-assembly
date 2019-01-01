package cli

import (
	"strconv"
	"strings"

	"github.com/fatih/color"
)

type cell struct {
	content  string
	colorDef *color.Color
}

// Table represents data as a table
type Table struct {
	currentRow int
	colSizes   []int
	data       [][]cell
	hasHeader  bool
}

var noColor = color.New()

// NewTable creates a new table
func NewTable() *Table {
	return &Table{
		currentRow: -1,
	}
}

// Header adds a header to table
func (t *Table) Header() *Table {
	t.hasHeader = true
	return t.Row()
}

// Row switches table to a new raw. All the Cell calls will cause new cells to
// be added to a new row
func (t *Table) Row() *Table {
	t.currentRow++
	t.data = append(t.data, make([]cell, 0))
	return t
}

// Cell adds cell
func (t *Table) Cell(s string) *Table {
	return t.ColorizedCell(s, noColor)
}

func (t *Table) ColorizedCell(s string, colorDef *color.Color) *Table {
	currentCell := len(t.data[t.currentRow])

	if currentCell >= len(t.colSizes) {
		t.colSizes = append(t.colSizes, 0)
	}

	if len(s) > t.colSizes[currentCell] {
		t.colSizes[currentCell] = len(s)
	}

	t.data[t.currentRow] = append(t.data[t.currentRow], cell{content: s, colorDef: colorDef})

	return t
}

// Render renders the table
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
			nextCell := cell{colorDef: noColor}

			if c < len(row) {
				nextCell = row[c]
			}

			renderedCells[c] = nextCell.colorDef.Sprintf("%-"+strconv.Itoa(size)+"s", nextCell.content)
		}
		renderedRows = append(renderedRows, "| "+strings.Join(renderedCells, " | ")+" |")

		if r == 0 && t.hasHeader {
			renderedRows = append(renderedRows, border)
		}
	}

	renderedRows = append(renderedRows, border)

	return strings.Join(renderedRows, "\n") + "\n"
}
