package main

import (
	"pangolin/pkg/cli"
	"pangolin/pkg/parser"
	"pangolin/pkg/path"
	"pangolin/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	pathMgr := path.NewPathManager()
 	jboxCli := cli.NewJboxClient()
 	parser := parser.NewPipeParser(pathMgr, jboxCli)
 	model := tui.NewTUI(pathMgr, parser, jboxCli)
 	p := tea.NewProgram(model, tea.WithAltScreen())
 	model.(*tui.TUI).SetProgram(p)
 	if _, err := p.Run(); err != nil {
		panic(err)
	}
}
