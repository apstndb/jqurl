package main

import (
	"golang.org/x/sync/errgroup"
	"io"
	"os"
	"os/exec"
)

type Executable interface {
	Wait() error
	Start() error
	SetStdin(io.Reader)
	SetStdout(io.Writer)
	SetStderr(io.Writer)
}

type CmdWrapper struct {
	*exec.Cmd
}

func (c *CmdWrapper) SetStderr(writer io.Writer) {
	c.Stderr = writer
}

func (c *CmdWrapper) SetStdin(reader io.Reader) {
	c.Cmd.Stdin = reader
}

func (c *CmdWrapper) SetStdout(writer io.Writer) {
	c.Cmd.Stdout = writer
}

type PipedCmds struct {
	left, right Executable
	writer      io.Writer
}

func (c *PipedCmds) SetStderr(writer io.Writer) {
	c.left.SetStderr(writer)
	c.right.SetStderr(writer)
}

func (c *PipedCmds) SetStdin(reader io.Reader) {
	c.left.SetStdin(reader)
}

func (c *PipedCmds) SetStdout(writer io.Writer) {
	c.right.SetStdout(writer)
}

func Join(left, right Executable) Executable {
	if left == nil {
		return right
	}
	if right == nil {
		return left
	}

	r, w := io.Pipe()

	left.SetStdout(w)
	right.SetStdin(r)

	return &PipedCmds{left, right, w}
}

func Cmd(c *exec.Cmd) Executable {
	return &CmdWrapper{c}
}

func (c *PipedCmds) Wait() error {
	var eg errgroup.Group
	eg.Go(func() error {
		if w, ok := c.writer.(io.WriteCloser); ok {
			defer w.Close()
		}
		return c.left.Wait()
	})
	eg.Go(func() error {
		return c.right.Wait()
	})
	return eg.Wait()
}

func (c *PipedCmds) Start() error {
	var eg errgroup.Group
	eg.Go(func() error {
		return c.left.Start()
	})
	eg.Go(func() error {
		return c.right.Start()
	})

	return eg.Wait()
}

var _ Executable = (*CmdWrapper)(nil)
var _ Executable = (*PipedCmds)(nil)

func SetAllToStd(e Executable) {
	e.SetStdin(os.Stdin)
	e.SetStdout(os.Stdout)
	e.SetStderr(os.Stderr)
}
