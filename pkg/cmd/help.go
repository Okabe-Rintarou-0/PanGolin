package cmd

import (
	"fmt"
	"io"
	"pangolin/pkg/cmd/models"
	"strings"
)

type HelpCommand struct{}

func NewHelpCommand() *HelpCommand {
	return &HelpCommand{}
}

func (h *HelpCommand) Execute(in io.Reader, out io.Writer) error {
	cmds := []Command{
		&HelpCommand{},
		&LsCommand{},
		&CdCommand{},
		&CpCommand{},
	}

	fmt.Fprint(out, "Commands:\n\n")
	for _, c := range cmds {
		fmt.Fprintf(out, "  %-10s %s\n", c.Name()+":", c.Help())
		if ex := c.Examples(); ex != "" {
			for line := range strings.SplitSeq(strings.TrimSuffix(ex, "\n"), "\n") {
				fmt.Fprintf(out, "    %s\n", line)
			}
		}
		fmt.Fprint(out, "\n")
	}

	fmt.Fprint(out, "Keybinds:\n  Enter          Execute command\n  Ctrl+C / Esc   Quit\n  Tab            Auto-complete\n  ↑/↓            Command history\n")
	return nil
}

func (h *HelpCommand) Hint(args []string) []models.HintEntry { return nil }
func (h *HelpCommand) Name() string                          { return "help" }
func (h *HelpCommand) Help() string                          { return "Show this help message" }
func (h *HelpCommand) Examples() string                      { return "" }
func (h *HelpCommand) ShouldExecAsync() bool                 { return false }
