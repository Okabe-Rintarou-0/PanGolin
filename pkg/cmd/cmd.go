package cmd

import (
	"io"
)

type Command interface {
	Execute(in io.Reader, out io.Writer) error
 	// Hint returns tab-completion candidates for the given input line.
 	// Return nil if no completions are available.
 	Hint(input string) []string
}
