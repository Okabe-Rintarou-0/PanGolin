package tui

import (
	"bytes"
	"io"
	"os"
	"pangolin/pkg/cli"
	"pangolin/pkg/cmd"
	"pangolin/pkg/cmd/models"
	"pangolin/pkg/parser"
	"pangolin/pkg/path"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	qrterminal "github.com/mdp/qrterminal/v3"
)

type mode int

const (
	modeLogin mode = iota
	modeShell
	headerHeight = 8
	inputHeight  = 3
)

var (
	brand    = lipgloss.Color("#6EE7B7")
	errorClr = lipgloss.Color("#F87171")
	mutedClr = lipgloss.Color("#4B5563")

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(brand).
			Padding(0, 2)

	titleStyle = lipgloss.NewStyle().
			Foreground(brand).
			Bold(true)

	shellStyle = lipgloss.NewStyle().Padding(0, 2)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorClr)

	accentText = lipgloss.NewStyle().
			Foreground(brand)

	mutedText = lipgloss.NewStyle().
			Foreground(mutedClr)
)

type msgQRReady struct {
	url string
}

type msgLoginSuccess struct{}

type msgLoginError struct {
	err string
}

type msgCheckingSession struct{}

type cpProgressMsg struct {
	completed int
	total     int
}

type cpDoneMsg struct {
	err error
}

type TUI struct {
	program         *tea.Program
	mode            mode
	loginError      string
	qrContent       string
	checkingSession bool
	jbox            cli.JboxClient

	info     []string
	lines    []string
	input    textinput.Model
	viewport viewport.Model
	width    int
	height   int
	pathMgr  path.PathManager
	parser   parser.Parser

	cpProgress    *progress.Model
	cpRunning     bool
	cpProgressIdx int

	history    []string
	historyIdx int
}

func NewTUI(pathMgr path.PathManager, parser parser.Parser, jbox cli.JboxClient) tea.Model {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Prompt = ""
	ti.Focus()

	vp := viewport.New(100, 20)
	p := progress.New(progress.WithDefaultGradient())

	return &TUI{
		mode: modeLogin,
		jbox: jbox,
		info: []string{
			"User: jAccount User",
			"Session: active",
			"Node: pangolin-core",
		},
		lines: []string{
			"🦔 pangolin shell ready",
			"type help for commands",
		},
		input:         ti,
		viewport:      vp,
		parser:        parser,
		pathMgr:       pathMgr,
		cpProgress:    &p,
		cpProgressIdx: -1,
		historyIdx:    -1,
	}
}

func (t *TUI) SetProgram(p *tea.Program) {
	t.program = p
}

func (t *TUI) Init() tea.Cmd {
	go func() {
		if t.jbox.HasSession() {
			t.program.Send(msgCheckingSession{})
		}

		err := t.jbox.Login(func(qrUrl string) {
			if t.program != nil {
				t.program.Send(msgQRReady{url: qrUrl})
			}
		})
		if err != nil {
			if t.program != nil {
				t.program.Send(msgLoginError{err: err.Error()})
			}
		} else {
			if t.program != nil {
				t.program.Send(msgLoginSuccess{})
			}
		}
	}()
	return textinput.Blink
}

func (t *TUI) renderQR(text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	var sb strings.Builder
	qrterminal.GenerateHalfBlock(text, qrterminal.L, &sb)
	return sb.String()
}

func (t *TUI) promptStr() string {
	return "🦔 " + t.pathMgr.CurrentPath().Path() + " > "
}

func (t *TUI) completePath() {
	inputVal := t.input.Value()
	if inputVal == "" || strings.Contains(inputVal, "|") {
		return
	}

	hasTrailingSpace := strings.HasSuffix(inputVal, " ")
	fields := strings.Fields(inputVal)
	if len(fields) == 0 {
		return
	}
	cmdName := fields[0]
	cmdArgs := fields[1:]

	var command cmd.Command
	switch cmdName {
	case "cd":
		command = cmd.NewCdCommand(t.pathMgr, t.jbox, cmdArgs...)
	case "ls":
		command = cmd.NewLsCommand(t.pathMgr, t.jbox, cmdArgs...)
	case "cp":
		command = cmd.NewCpCommand(t.pathMgr, t.jbox, nil, cmdArgs...)
	default:
		return
	}

	hints := command.Hint(cmdArgs)
	if len(hints) == 0 {
		t.lines = append(t.lines, mutedText.Render("  (no completions)"))
		t.syncViewPort()
		t.viewport.GotoBottom()
		return
	}

	var prefix strings.Builder
	prefix.WriteString(cmdName)
	lastArgIdx := len(cmdArgs)
	if !hasTrailingSpace && len(cmdArgs) > 0 {
		lastArgIdx = len(cmdArgs) - 1
	}
	for i := 0; i < lastArgIdx; i++ {
		prefix.WriteString(" ")
		prefix.WriteString(cmdArgs[i])
	}
	prefix.WriteString(" ")

	if len(hints) == 1 {
		prefix.WriteString(hints[0].RealValue())
		t.input.SetValue(prefix.String())
		t.input.CursorEnd()
		return
	}

	currentPartial := ""
	if !hasTrailingSpace && len(cmdArgs) > 0 {
		currentPartial = cmdArgs[len(cmdArgs)-1]
	}

	commonPrefix := longestCommonPrefix(hints)
	if commonPrefix != currentPartial && commonPrefix != "" {
		prefix.WriteString(commonPrefix)
		t.input.SetValue(prefix.String())
		t.input.CursorEnd()
		return
	}

	var sb strings.Builder
	for i, hint := range hints {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(hint.DisplayValue())
	}
	t.lines = append(t.lines, "  "+sb.String())

	t.syncViewPort()
	t.viewport.GotoBottom()
}

func longestCommonPrefix(hints models.HintEntries) string {
	if len(hints) == 0 {
		return ""
	}
	prefix := hints[0].RealValue()
	for _, hint := range hints[1:] {
		for len(prefix) > 0 && !strings.HasPrefix(hint.RealValue(), prefix) {
			prefix = prefix[:len(prefix)-1]
		}
	}
	return prefix
}

func (t *TUI) exec(inputLine string) {
	inputLine = strings.TrimSpace(inputLine)
	if inputLine == "" {
		return
	}

	if len(t.history) == 0 || t.history[len(t.history)-1] != inputLine {
		t.history = append(t.history, inputLine)
	}
	t.historyIdx = -1

	t.lines = append(t.lines, t.promptStr()+inputLine)

	if inputLine == "clear" {
		t.lines = []string{}
		t.syncViewPort()
		t.viewport.GotoBottom()
		return
	}

	// optimize it
	parts := strings.Fields(inputLine)
	if len(parts) > 0 && parts[0] == "cp" {
		cpCmd := cmd.NewCpCommand(t.pathMgr, t.jbox, func(current, total int) {
			if t.program != nil {
				t.program.Send(cpProgressMsg{current, total})
			}
		}, parts[1:]...)
		t.execAsync(cpCmd)
		return
	}

	command := t.parser.Parse(inputLine)
	if command == nil {
		return
	}

	var outputBuffer bytes.Buffer
	err := command.Execute(os.Stdin, &outputBuffer)
	if err != nil {
		t.lines = append(t.lines, errorStyle.Render("Error: "+err.Error()))
	}

	outputStr := outputBuffer.String()
	if outputStr != "" {
		outputStr = strings.ReplaceAll(outputStr, "\r", "")
		outputLines := strings.SplitSeq(strings.TrimRight(outputStr, "\n"), "\n")
		for line := range outputLines {
			if line != "" {
				t.lines = append(t.lines, line)
			}
		}
	}

	t.syncViewPort()
	t.viewport.GotoBottom()
}

func (t *TUI) execAsync(c cmd.Command) {
	lineIdx := len(t.lines)
	t.lines = append(t.lines, mutedText.Render("Counting..."))
	t.cpProgressIdx = lineIdx
	t.cpRunning = true

	go func() {
		err := c.Execute(nil, io.Discard)
		if t.program != nil {
			t.program.Send(cpDoneMsg{err})
		}
	}()
}

func (t *TUI) layoutViewport() {
	if t.width == 0 || t.height == 0 {
		return
	}
	t.viewport.Width = t.width - 2
	t.viewport.Height = max(t.height-headerHeight-inputHeight, 3)
}

func (t *TUI) syncViewPort() {
	var sb strings.Builder
	for _, l := range t.lines {
		sb.WriteString(l)
		sb.WriteString("\n")
	}
	t.viewport.SetContent(shellStyle.Width(t.width - 2).Render(sb.String()))
}

func (t *TUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case msgQRReady:
		t.qrContent = t.renderQR(msg.url)
		t.checkingSession = false
		return t, nil

	case msgLoginSuccess:
		t.mode = modeShell
		t.info = t.jbox.SessionInfo()
		t.layoutViewport()
		t.syncViewPort()
		return t, nil

	case msgLoginError:
		t.loginError = msg.err
		t.checkingSession = false
		return t, nil

	case msgCheckingSession:
		t.checkingSession = true
		return t, nil

	case cpProgressMsg:
		if t.cpRunning && t.cpProgressIdx >= 0 && t.cpProgressIdx < len(t.lines) && msg.total > 0 {
			t.cpProgress.Width = t.width - 4
			pct := float64(msg.completed) / float64(msg.total)
			bar := t.cpProgress.ViewAs(pct)
			t.lines[t.cpProgressIdx] = bar
			t.syncViewPort()
			t.viewport.GotoBottom()
		}
		return t, nil

	case cpDoneMsg:
		t.cpRunning = false
		if t.cpProgressIdx >= 0 && t.cpProgressIdx < len(t.lines) {
			if msg.err != nil {
				t.lines[t.cpProgressIdx] = errorStyle.Render("Error: " + msg.err.Error())
			} else {
				t.lines[t.cpProgressIdx] = "Done"
			}
		}
		t.cpProgressIdx = -1
		t.syncViewPort()
		t.viewport.GotoBottom()
		return t, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return t, tea.Quit
		case tea.KeyEnter:
			if t.mode == modeShell {
				t.exec(t.input.Value())
				t.input.SetValue("")
			}
		case tea.KeyUp:
			if t.mode == modeShell && len(t.history) > 0 {
				if t.historyIdx == -1 {
					t.historyIdx = len(t.history) - 1
				} else if t.historyIdx > 0 {
					t.historyIdx--
				}
				t.input.SetValue(t.history[t.historyIdx])
				t.input.CursorEnd()
			}
		case tea.KeyDown:
			if t.mode == modeShell {
				if t.historyIdx >= 0 {
					t.historyIdx++
					if t.historyIdx >= len(t.history) {
						t.historyIdx = -1
						t.input.SetValue("")
					} else {
						t.input.SetValue(t.history[t.historyIdx])
					}
					t.input.CursorEnd()
				}
			}
		case tea.KeyTab:
			if t.mode == modeShell {
				t.completePath()
			}
		}

	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
		t.layoutViewport()
		t.syncViewPort()
	}

	if t.mode == modeShell {
		t.input, cmd = t.input.Update(msg)
		cmds = append(cmds, cmd)

		t.viewport, cmd = t.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return t, tea.Batch(cmds...)
}

func (t *TUI) View() string {
	if t.width == 0 || t.height == 0 {
		return "Loading..."
	}

	if t.mode == modeLogin {
		return t.loginView()
	}

	return t.shellView()
}

func (t *TUI) shellView() string {
	infoBox := boxStyle.Copy().Width(t.width - 2)
	info := strings.Join(t.info, "\n")
	header := infoBox.Render(titleStyle.Render("🦔 PANGOLIN SESSION INFO") + "\n\n" + info)

	input := boxStyle.Copy().Width(t.width - 2).Render(t.promptStr() + t.input.View())

	result := lipgloss.JoinVertical(lipgloss.Left, header, t.viewport.View(), input)
	return result
}

func (t *TUI) loginView() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(titleStyle.Render("  🦔 PANGOLIN Login"))
	sb.WriteString("\n\n")
	if !t.checkingSession {
		sb.WriteString(accentText.Render("  Scan the QR code with WeChat or SJTU App"))
		sb.WriteString("\n\n")
	}

	if t.qrContent != "" {
		sb.WriteString(t.qrContent)
		sb.WriteString("\n")
	} else if t.checkingSession {
		sb.WriteString("  " + mutedText.Render("📦 Checking session...") + "\n\n")
	} else if t.loginError != "" {
		sb.WriteString("  " + errorStyle.Render("✖ "+t.loginError) + "\n\n")
	} else {
		sb.WriteString("  " + mutedText.Render("⟳ Loading QR code...") + "\n\n")
	}

	sb.WriteString("\n  " + mutedText.Render("Ctrl+C to quit"))

	innerW := max(t.width-12, 30)
	minH := 12

	loginBox := boxStyle.Copy().
		Width(innerW).
		Height(minH).
		Align(lipgloss.Left)

	return lipgloss.Place(
		t.width, t.height,
		lipgloss.Center, lipgloss.Center,
		loginBox.Render(sb.String()),
	)
}
