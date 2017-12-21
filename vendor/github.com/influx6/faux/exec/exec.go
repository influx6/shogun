package exec

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"context"

	"github.com/influx6/faux/metrics"
)

// errors ...
var (
	ErrCommandFailed = errors.New("Command failed to execute succcesfully")
)

// CommanderOption defines a function type that aguments a commander's field.
type CommanderOption func(*Commander)

// Command sets the command for the Commander.
func Command(c string, m ...interface{}) CommanderOption {
	return func(cm *Commander) {
		cm.Command = fmt.Sprintf(c, m...)
	}
}

// Commands sets the subcommands for the Commander exec call.
// If subcommands are set then the Binary, Flag and Command are ignored
// and the values of the subcommand is used.
func Commands(p ...string) CommanderOption {
	return func(cm *Commander) {
		cm.SubCommands = p
	}
}

// Dir sets the Directory for the Commander exec call.
func Dir(p string) CommanderOption {
	return func(cm *Commander) {
		cm.Dir = p
	}
}

// Binary sets the binary command for the Commander.
func Binary(bin string, flag string) CommanderOption {
	return func(cm *Commander) {
		cm.Binary = bin
		cm.Flag = flag
	}
}

// Sync sets the commander to run in synchronouse mode.
func Sync() CommanderOption {
	return SetAsync(false)
}

// Async sets the commander to run in asynchronouse mode.
func Async() CommanderOption {
	return SetAsync(true)
}

// SetAsync sets the command for the Commander.
func SetAsync(b bool) CommanderOption {
	return func(cm *Commander) {
		cm.Async = b
	}
}

// Input sets the input reader for the Commander.
func Input(in io.Reader) CommanderOption {
	return func(cm *Commander) {
		cm.In = in
	}
}

// Output sets the output writer for the Commander.
func Output(out io.Writer) CommanderOption {
	return func(cm *Commander) {
		cm.Out = out
	}
}

// Err sets the error writer for the Commander.
func Err(err io.Writer) CommanderOption {
	return func(cm *Commander) {
		cm.Err = err
	}
}

// Envs sets the map of environment for the Commander.
func Envs(envs map[string]string) CommanderOption {
	return func(cm *Commander) {
		cm.Envs = envs
	}
}

// Apply takes the giving series of CommandOption returning a function that always applies them to passed in commanders.
func Apply(ops ...CommanderOption) CommanderOption {
	return func(cm *Commander) {
		for _, op := range ops {
			op(cm)
		}
	}
}

// ApplyImmediate applies the options immediately to the Commander.
func ApplyImmediate(cm *Commander, ops ...CommanderOption) *Commander {
	for _, op := range ops {
		op(cm)
	}

	return cm
}

// Commander runs provided command within a /bin/sh -c "{COMMAND}", returning
// response associatedly. It also attaches if provided stdin, stdout and stderr readers/writers.
// Commander allows you to set the binary to use and flag, where each defaults to /bin/sh for binary
// and -c for flag respectively.
type Commander struct {
	Async       bool
	Command     string
	SubCommands []string
	Dir         string
	Binary      string
	Flag        string
	Envs        map[string]string
	In          io.Reader
	Out         io.Writer
	Err         io.Writer
}

// New returns a new Commander instance.
func New(ops ...CommanderOption) *Commander {
	cm := new(Commander)

	for _, op := range ops {
		op(cm)
	}

	return cm
}

// ExecWithExitCode executes giving command associated within the command with os/exec.
func (c *Commander) ExecWithExitCode(ctx context.Context, metric metrics.Metrics) (int, error) {
	if err := c.Exec(ctx, metric); err != nil {
		return getExitStatus(err), err
	}

	return 0, nil
}

// Exec executes giving command associated within the command with os/exec.
func (c *Commander) Exec(ctx context.Context, metric metrics.Metrics) error {
	if c.Binary == "" {
		c.Binary = "/bin/sh"
	}

	if c.Flag == "" {
		c.Flag = "-c"
	}

	var execCommand []string

	switch {
	case c.Command == "" && len(c.SubCommands) != 0:
		execCommand = c.SubCommands
	case c.Command == "" && len(c.SubCommands) == 0:
		execCommand = append(execCommand, c.Binary)
	case c.Command != "" && len(c.SubCommands) == 0:
		execCommand = append(execCommand, c.Binary, c.Flag, c.Command)
	}

	var errCopy bytes.Buffer
	var multiErr io.Writer

	if c.Err != nil {
		multiErr = io.MultiWriter(&errCopy, c.Err)
	} else {
		multiErr = &errCopy
	}

	cmder := exec.Command(execCommand[0], execCommand[1:]...)
	cmder.Dir = c.Dir
	cmder.Stderr = multiErr
	cmder.Stdin = c.In
	cmder.Stdout = c.Out
	cmder.Env = os.Environ()

	if c.Envs != nil {
		for name, val := range c.Envs {
			cmder.Env = append(cmder.Env, fmt.Sprintf("%s=%s", name, val))
		}
	}

	metric.Emit(metrics.Info("Executing native commands"), metrics.WithID("shell:exec"), metrics.WithFields(metrics.Field{
		"command": strings.Join(execCommand, " "),
		"envs":    c.Envs,
	}))

	if !c.Async {
		err := cmder.Run()
		if err != nil {
			metric.Emit(metrics.Error(err), metrics.WithID("shell:exec:error"), metrics.WithFields(metrics.Field{
				"error":      err.Error(),
				"command":    strings.Join(execCommand, " "),
				"envs":       c.Envs,
				"error_data": string(errCopy.Bytes()),
			}))
		}
		return err
	}

	if err := cmder.Start(); err != nil {
		metric.Emit(metrics.Error(err), metrics.WithID("shell:exec:error"), metrics.WithFields(metrics.Field{
			"envs":       c.Envs,
			"error":      err.Error(),
			"command":    strings.Join(execCommand, " "),
			"error_data": string(errCopy.Bytes()),
		}))
		return err
	}

	go func() {
		<-ctx.Done()
		if cmder.Process == nil {
			return
		}

		cmder.Process.Kill()
	}()

	if err := cmder.Wait(); err != nil {
		metric.Emit(metrics.Error(err), metrics.WithID("shell:exec:error"), metrics.WithFields(metrics.Field{
			"envs":       c.Envs,
			"error":      err.Error(),
			"command":    strings.Join(execCommand, " "),
			"error_data": string(errCopy.Bytes()),
		}))
		return err
	}

	if cmder.ProcessState == nil {
		return nil
	}

	if !cmder.ProcessState.Success() {
		metric.Emit(metrics.Error(ErrCommandFailed), metrics.WithID("shell:exec:error"), metrics.WithFields(metrics.Field{
			"envs":       c.Envs,
			"error":      ErrCommandFailed.Error(),
			"command":    strings.Join(execCommand, " "),
			"error_data": string(errCopy.Bytes()),
		}))
		return ErrCommandFailed
	}

	return nil
}

type exitStatus interface {
	ExitStatus() int
}

func getExitStatus(err error) int {
	if err == nil {
		return 0
	}
	if e, ok := err.(exitStatus); ok {
		return e.ExitStatus()
	}
	if e, ok := err.(*exec.ExitError); ok {
		if ex, ok := e.Sys().(exitStatus); ok {
			return ex.ExitStatus()
		}
	}
	return 1
}
