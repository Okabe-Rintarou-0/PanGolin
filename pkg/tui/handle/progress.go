package handle

import (
	"pangolin/pkg/tui/msg"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

type ProgressBarHandle struct {
	lineIdx int
	current int
	total   int

	program *tea.Program
	pbar    *progress.Model
	err     error
}

func NewProgressBarHandle(lineIdx int, program *tea.Program) *ProgressBarHandle {
	return &ProgressBarHandle{
		lineIdx: lineIdx,
		total:   0,
		current: 0,
		program: program,
	}
}

func (h *ProgressBarHandle) Create() {
	pbar := progress.New(progress.WithDefaultGradient())
	h.pbar = &pbar
}

func (h *ProgressBarHandle) Set(current int, total int) {
	h.current = current
	h.total = total

	h.program.Send(msg.ProgressMsg{
		LineIdx: h.lineIdx,
		Current: h.current,
		Total:   h.total,
		Pbar:    h.pbar,
		Err:     nil,
	})
}

func (h *ProgressBarHandle) SetError(err error) {
	h.program.Send(msg.ProgressMsg{
		LineIdx: h.lineIdx,
		Current: h.current,
		Total:   h.total,
		Pbar:    h.pbar,
		Err:     err,
	})
}
