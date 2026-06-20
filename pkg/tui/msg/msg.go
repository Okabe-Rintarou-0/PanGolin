package msg

import "github.com/charmbracelet/bubbles/progress"

type QRReadyMsg struct {
	Url string
}

type LoginSuccessMsg struct{}

type LoginErrorMsg struct {
	Err string
}

type CheckingSessionMsg struct{}

type ProgressMsg struct {
	Current int
	Total   int
	LineIdx int
	Pbar    *progress.Model
	Err     error
}

type ExecuteDoneMsg struct{}
