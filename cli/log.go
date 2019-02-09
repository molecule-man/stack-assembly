package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/molecule-man/stack-assembly/cli/color"
)

var Output io.Writer = os.Stdout

func Fprint(w io.Writer, msg string) {
	fmt.Fprintln(w, msg)
}

func Print(msg string) {
	Fprint(Output, msg)
}

func Printf(msg string, args ...interface{}) {
	Print(fmt.Sprintf(msg, args...))
}

func Error(msg string) {
	Print(color.Fail(msg))
}

func Errorf(format string, args ...interface{}) {
	Printf(color.Fail(format), args...)
}

func Info(msg string) {
	Print(msg)
}

func Infof(format string, args ...interface{}) {
	Print(fmt.Sprintf(format, args...))
}

func Warn(msg string) {
	Print(color.Warn(msg))
}

func Warnf(format string, args ...interface{}) {
	Printf(color.Warn(format), args...)
}

type Logger struct {
	prefix string
}

func PrefixedLogger(prefix string) *Logger {
	return &Logger{
		prefix: prefix,
	}
}

func (l *Logger) prefixedMsg(msg string) string {
	return fmt.Sprintf("%s%s", l.prefix, msg)
}

func (l *Logger) Fprint(w io.Writer, msg string) {
	Fprint(w, l.prefixedMsg(msg))
}

func (l *Logger) Print(msg string) {
	Print(l.prefixedMsg(msg))
}

func (l *Logger) Error(msg string) {
	Error(l.prefixedMsg(msg))
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	Errorf(l.prefixedMsg(format), args...)
}

func (l *Logger) Info(msg string) {
	Info(l.prefixedMsg(msg))
}

func (l *Logger) Infof(format string, args ...interface{}) {
	Infof(l.prefixedMsg(format), args...)
}

func (l *Logger) Warn(msg string) {
	Warn(l.prefixedMsg(msg))
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	Warnf(l.prefixedMsg(format), args...)
}
