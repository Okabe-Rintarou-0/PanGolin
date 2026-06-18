package cmd

 import (
 	"fmt"
 	"io"
 )

type HelpCommand struct{}

func NewHelpCommand() *HelpCommand {
	return &HelpCommand{}
}

func (h *HelpCommand) Execute(in io.Reader, out io.Writer) error {
 	helpText := `Available commands:
   help           Show this help message
   ls [<path>]    List files and directories
   clear          Clear the screen
 `
 	helpText += `
 Keybinds:
   Enter          Execute command
   Ctrl+C / Esc   Quit
 `

 	_, err := fmt.Fprint(out, helpText)
 	return err
}
