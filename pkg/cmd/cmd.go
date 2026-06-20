package cmd

import (
	"io"
	"pangolin/pkg/cmd/models"
)

type Command interface {
	Execute(in io.Reader, out io.Writer) error
	Hint(args []string) []models.HintEntry
	Name() string
	Help() string
	Examples() string
}
