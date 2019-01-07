package cli

import (
	"fmt"

	"github.com/fatih/color"
)

func Print(msg string) {
	fmt.Println(msg)
}

func ColorPrint(col *color.Color, msg string) {
	fmt.Println(col.Sprint(msg))
}

func Error(msg string) {
	Print(FailureColor.Sprint(msg))
}

func Errorf(format string, args ...interface{}) {
	Print(FailureColor.Sprintf(format, args...))
}

func Info(msg string) {
	Print(msg)
}

func Infof(format string, args ...interface{}) {
	Print(fmt.Sprintf(format, args...))
}

func Warn(msg string) {
	Print(WarnColor.Sprint(msg))
}

func Warnf(format string, args ...interface{}) {
	Print(WarnColor.Sprintf(format, args...))
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

func (l *Logger) Print(msg string) {
	Print(l.prefixedMsg(msg))
}

func (l *Logger) ColorPrint(col *color.Color, msg string) {
	ColorPrint(col, l.prefixedMsg(msg))
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
