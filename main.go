package main

import (
	"pangolin/pkg/cli"
	"pangolin/pkg/parser"
	"pangolin/pkg/path"
	"pangolin/pkg/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	pathMgr := path.NewPathManager()
	jboxCli := cli.NewJboxClient()
	parser := parser.NewPipeParser(pathMgr, jboxCli)
	t := tui.NewTUI(pathMgr, parser, jboxCli)
	p := tea.NewProgram(t, tea.WithAltScreen())
	parser.Init(p)
	t.Start(p)
	if _, err := p.Run(); err != nil {
		panic(err)
	}
}
