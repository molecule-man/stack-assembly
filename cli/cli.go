package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

func Fprint(w io.Writer, msg string) {
	fmt.Fprintln(w, msg)
}

type CLI struct {
	Reader  io.Reader
	Writer  io.Writer
	Errorer io.Writer

	Color Color
}

func (cli CLI) Print(msg string) {
	Fprint(cli.Writer, msg)
}

func (cli CLI) Printf(msg string, args ...interface{}) {
	cli.Print(fmt.Sprintf(msg, args...))
}

func (cli CLI) Error(msg string) {
	Fprint(cli.Errorer, cli.Color.Fail(msg))
}

func (cli CLI) Errorf(format string, args ...interface{}) {
	cli.Error(fmt.Sprintf(format, args...))
}

func (cli CLI) Info(msg string) {
	cli.Print(msg)
}

func (cli CLI) Infof(format string, args ...interface{}) {
	cli.Print(fmt.Sprintf(format, args...))
}

func (cli CLI) Warn(msg string) {
	cli.Print(cli.Color.Warn(msg))
}

func (cli CLI) Warnf(format string, args ...interface{}) {
	cli.Printf(cli.Color.Warn(format), args...)
}

func (cli CLI) Prompt(commands []PromptCmd) error {
	cli.Print("*** Commands ***")

	for _, c := range commands {
		cli.Print("  " + c.Description)
	}

	response, err := cli.Ask("What now> ")
	if err != nil {
		return err
	}

	for _, c := range commands {
		for _, inp := range c.TriggerInputs {
			if inp == response {
				c.Action()
				return nil
			}
		}
	}

	cli.Warnf("Command %s is not known\n", response)

	return nil
}

func (cli CLI) Fask(w io.Writer, query string, args ...interface{}) (string, error) {
	fmt.Fprintf(w, query, args...)

	reader := bufio.NewReader(cli.Reader)
	response, err := reader.ReadString('\n')

	if err != nil {
		return "", fmt.Errorf("reading user input failed with err: %w", err)
	}

	return strings.TrimSpace(response), nil
}

func (cli CLI) Ask(query string, args ...interface{}) (string, error) {
	return cli.Fask(cli.Writer, query, args...)
}

func (cli CLI) PrefixedLogger(prefix string) *Logger {
	return &Logger{
		prefix: prefix,
		cli:    &cli,
	}
}

type Logger struct {
	prefix string
	cli    *CLI
}

func (l *Logger) prefixedMsg(msg string) string {
	return fmt.Sprintf("%s%s", l.prefix, msg)
}

func (l *Logger) Fprint(w io.Writer, msg string) {
	Fprint(w, l.prefixedMsg(msg))
}

func (l *Logger) Print(msg string) {
	l.cli.Print(l.prefixedMsg(msg))
}

func (l *Logger) Error(msg string) {
	l.cli.Error(l.prefixedMsg(msg))
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.cli.Errorf(l.prefixedMsg(format), args...)
}

func (l *Logger) Info(msg string) {
	l.cli.Info(l.prefixedMsg(msg))
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.cli.Infof(l.prefixedMsg(format), args...)
}

func (l *Logger) Warn(msg string) {
	l.cli.Warn(l.prefixedMsg(msg))
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.cli.Warnf(l.prefixedMsg(format), args...)
}

var ErrPromptCommandIsNotKnown = errors.New("prompt command is not known")

type PromptCmd struct {
	TriggerInputs []string
	Description   string
	Action        func()
}
