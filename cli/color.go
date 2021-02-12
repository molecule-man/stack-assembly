package cli

import (
	"bufio"
	"strconv"
	"strings"
)

var NoColor bool

type ColorCode int

const (
	// Foreground colors.
	BlackFG ColorCode = iota + 30
	RedFG
	GreenFG
	YellowFG
	BlueFG
	MagentaFG
	CyanFG
	WhiteFG

	Defaultt ColorCode = 39

	Reset ColorCode = 0
	Bold  ColorCode = 1

	start = "\033["
	end   = "m"
	sep   = ";"

	ResetCode = "\033[0m"
)

type Color struct {
	Disabled bool
}

func (c Color) Code(attr ColorCode, args ...ColorCode) string {
	if c.Disabled {
		return ""
	}

	attrs := []string{strconv.FormatInt(int64(attr), 10)}
	for _, i := range args {
		attrs = append(attrs, strconv.FormatInt(int64(i), 10))
	}

	return start + strings.Join(attrs, sep) + end
}

func (c Color) Black(msg string) string   { return c.Code(BlackFG) + msg + c.ColorReset() }
func (c Color) Red(msg string) string     { return c.Code(RedFG) + msg + c.ColorReset() }
func (c Color) Green(msg string) string   { return c.Code(GreenFG) + msg + c.ColorReset() }
func (c Color) Yellow(msg string) string  { return c.Code(YellowFG) + msg + c.ColorReset() }
func (c Color) Blue(msg string) string    { return c.Code(BlueFG) + msg + c.ColorReset() }
func (c Color) Magenta(msg string) string { return c.Code(MagentaFG) + msg + c.ColorReset() }
func (c Color) Cyan(msg string) string    { return c.Code(CyanFG) + msg + c.ColorReset() }
func (c Color) White(msg string) string   { return c.Code(WhiteFG) + msg + c.ColorReset() }

func (c Color) ColorSuccess() string { return c.Code(GreenFG) }
func (c Color) ColorFail() string    { return c.Code(RedFG, Bold) }
func (c Color) ColorNeutral() string { return c.Code(CyanFG) }
func (c Color) ColorWarn() string    { return c.Code(YellowFG) }
func (c Color) ColorReset() string   { return c.Code(Reset) }

func (c Color) Success(msg string) string {
	return c.ColorSuccess() + msg + c.ColorReset()
}

func (c Color) Fail(msg string) string {
	return c.ColorFail() + msg + c.ColorReset()
}

func (c Color) Neutral(msg string) string {
	return c.ColorNeutral() + msg + c.ColorReset()
}

func (c Color) Warn(msg string) string {
	return c.ColorWarn() + msg + c.ColorReset()
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
