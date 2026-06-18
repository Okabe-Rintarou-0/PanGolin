package cmd

import (
	"io"
)

type Command interface {
	Execute(in io.Reader, out io.Writer) error
}
