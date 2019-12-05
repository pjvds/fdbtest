package fdbtest

import (
	"fmt"
	"io"
)

type Logger interface {
	Log(v ...interface{})
	Logf(format string, v ...interface{})
}

type WriterLogger struct {
	io.Writer
}

func (s WriterLogger) Log(v ...interface{}) {
	fmt.Fprint(s, v...)
}

func (s WriterLogger) Logf(format string, v ...interface{}) {
	fmt.Fprintf(s, format, v...)
}

type NilLogger struct{}

func (n *NilLogger) Log(v ...interface{}) {
}

func (n *NilLogger) Logf(format string, v ...interface{}) {
}
