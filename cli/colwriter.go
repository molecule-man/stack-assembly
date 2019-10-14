package cli

import (
	"bytes"
	"io"
	"strings"
	"unicode/utf8"
)

type colWriterLine []string

type ColWriter struct {
	PadLastColumn  bool
	SprintLineFunc func([]string, []int) string
	DecorateLine   func(string) string

	output io.Writer
	sep    string

	widths []int
	buf    *bytes.Buffer
}

func (cw *ColWriter) Write(buf []byte) (n int, err error) {
	return cw.buf.Write(buf)
}

func (cw *ColWriter) Flush() error {
	if cw.buf.Len() == 0 {
		return nil
	}

	lastBytes := []byte{}

	buf := cw.buf.Bytes()
	if buf[len(buf)-1] == '\n' {
		lastBytes = []byte{'\n'}
		buf = buf[:len(buf)-1]
	}

	ll := strings.Split(string(buf), "\n")
	lines := make([]colWriterLine, len(ll))

	for i, l := range ll {
		cells := strings.Split(l, "\t")

		for j, cell := range cells {
			cw.updateWidth(j, cell)
		}

		lines[i] = cells
	}

	outputLines := make([]string, len(lines))
	for i, line := range lines {
		outputLines[i] = cw.sprintLine(line)
	}

	output := strings.Join(outputLines, "\n")
	buf = []byte(output)
	buf = append(buf, lastBytes...)
	_, err := cw.output.Write(buf)

	cw.buf = bytes.NewBuffer([]byte{})

	return err
}

func (cw *ColWriter) sprintLine(line []string) string {
	if cw.SprintLineFunc != nil {
		output := cw.SprintLineFunc(line, cw.widths)

		if output != "" {
			return output
		}
	}

	cells := make([]string, len(cw.widths))

	for i := range cw.widths {
		c := ""
		if i < len(line) {
			c = line[i]
		}

		cells[i] = cw.sprintCell(i, c)
	}

	output := strings.Join(cells, cw.sep)

	if !cw.PadLastColumn {
		output = strings.TrimRight(output, " ")
	}

	if cw.DecorateLine != nil {
		output = cw.DecorateLine(output)
	}

	return output
}

func (cw *ColWriter) sprintCell(n int, content string) string {
	contentWidth := cw.width(content)

	if HasColors(content) && !strings.HasSuffix(content, ResetCode) {
		content += ResetCode
	}

	for i := 0; i < cw.widths[n]-contentWidth; i++ {
		content += " "
	}

	return content
}

func (cw *ColWriter) updateWidth(i int, cell string) {
	for len(cw.widths) <= i {
		cw.widths = append(cw.widths, 0)
	}

	width := cw.width(cell)
	if width > cw.widths[i] {
		cw.widths[i] = width
	}
}

func (cw *ColWriter) width(cell string) int {
	return utf8.RuneCount([]byte(RmColors(cell)))
}

func NewColWriter(w io.Writer, sep string) *ColWriter {
	return &ColWriter{
		output: w,
		sep:    sep,
		widths: []int{},
		buf:    bytes.NewBuffer([]byte{}),
	}
}
