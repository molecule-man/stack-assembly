package color

import (
	"bufio"
	"strconv"
	"strings"
)

var NoColor bool

type Color int

const (
	// Foreground colors
	Black Color = iota + 30
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White

	Defaultt Color = 39

	Reset Color = 0
	Bold  Color = 1

	start = "\033["
	end   = "m"
	sep   = ";"
)

var (
	CodeSuccess = CodeFunc(Green)
	CodeFail    = CodeFunc(Red, Bold)
	CodeNeutral = CodeFunc(Cyan)
	CodeWarn    = CodeFunc(Yellow)
	CodeReset   = CodeFunc(Reset)
)

func Code(attr Color, args ...Color) string {
	if NoColor {
		return ""
	}

	attrs := []string{strconv.FormatInt(int64(attr), 10)}
	for _, i := range args {
		attrs = append(attrs, strconv.FormatInt(int64(i), 10))
	}

	return start + strings.Join(attrs, sep) + end
}

func CodeFunc(attr Color, args ...Color) func() string {
	return func() string {
		return Code(attr, args...)
	}
}

func Success(msg string) string {
	return CodeSuccess() + msg + CodeReset()
}

func Fail(msg string) string {
	return CodeFail() + msg + CodeReset()
}

func Neutral(msg string) string {
	return CodeNeutral() + msg + CodeReset()
}

func Warn(msg string) string {
	return CodeWarn() + msg + CodeReset()
}

func HasColors(str string) bool {
	return strings.Contains(str, start)
}

func RmColors(str string) string {
	scanner := bufio.NewScanner(strings.NewReader(str))
	onColor := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		colorStarted := false
		colorStartedAt := 0

		for i := 0; i < len(data); i++ {
			if i > 0 && data[i-1] == '\033' && data[i] == '[' {
				colorStarted = true
				colorStartedAt = i - 1
			} else if colorStarted && data[i] == 'm' {
				return i + 1, data[:colorStartedAt], nil
			}
		}

		return 0, data, bufio.ErrFinalToken
	}

	result := ""

	scanner.Split(onColor)
	for scanner.Scan() {
		result += scanner.Text()
	}
	return result
}
