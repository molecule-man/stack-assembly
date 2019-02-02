package conf

import (
	"errors"
	"fmt"
	"os/exec"
)

type hookCmd []string

type HookCmds []hookCmd

type HookError struct {
	cmd hookCmd
	err error
	out []byte
}

func (e *HookError) Error() string {
	return fmt.Sprintf("hook command %v failed with err: %v, output:\n%s", e.cmd, e.err, string(e.out))
}

func (h HookCmds) Exec() error {
	for _, hc := range h {
		err := hc.exec()
		if err != nil {
			return err
		}
	}
	return nil
}

func (h hookCmd) exec() error {

	if len(h) == 0 {
		return errors.New("hook command is empty")
	}
	cmd := h[0]
	out, err := exec.Command(cmd, h[1:]...).CombinedOutput()

	if err != nil {
		return &HookError{h, err, out}
	}
	return nil
}
