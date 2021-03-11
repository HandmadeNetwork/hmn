package oops

import (
	"fmt"

	"github.com/go-stack/stack"
	"github.com/rs/zerolog"
)

type Error struct {
	Message string
	Wrapped error
	Stack   CallStack
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %v", e.Message, e.Wrapped)
}

func (e *Error) Unwrap() error {
	return e.Wrapped
}

type CallStack []StackFrame

func (s CallStack) MarshalZerologArray(a *zerolog.Array) {
	for _, frame := range s {
		a.Object(frame)
	}
}

type StackFrame struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Function string `json:"function"`
}

func (f StackFrame) MarshalZerologObject(e *zerolog.Event) {
	e.
		Str("file", f.File).
		Int("line", f.Line).
		Str("function", f.Function)
}

var ZerologStackMarshaler = func(err error) interface{} {
	if asOops, ok := err.(*Error); ok {
		return asOops.Stack
	}
	return nil
}

func New(wrapped error, format string, args ...interface{}) error {
	trace := stack.Trace().TrimRuntime()
	frames := make(CallStack, len(trace))
	for i, call := range trace {
		callFrame := call.Frame()
		frames[i] = StackFrame{
			File:     callFrame.File,
			Line:     callFrame.Line,
			Function: callFrame.Function,
		}
	}

	return &Error{
		Message: fmt.Sprintf(format, args...),
		Wrapped: wrapped,
		Stack:   frames,
	}
}
