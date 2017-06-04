package main

import (
	"io"
	"os"
)

var f Format

type Format struct {
	Stdin  io.Reader // os.Stdin if nil
	Stdout io.Writer // os.Stdout if nil
	Stderr io.Writer // os.Stderr if nil
}

func (f *Format) Refmt(in, out string) error {
	return nil
}

func (f *Format) stdin() io.Reader {
	if f.Stdin != nil {
		return f.Stdin
	}
	return os.Stdin
}

func (f *Format) stdout() io.Writer {
	if f.Stdout != nil {
		return f.Stdout
	}
	return os.Stdout
}

func (f *Format) stderr() io.Writer {
	if f.Stderr != nil {
		return f.Stderr
	}
	return os.Stderr
}

func Refmt(in, out string) error { return f.Refmt(in, out) }
