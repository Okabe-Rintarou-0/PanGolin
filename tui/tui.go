package tui

import (
	"bytes"
	"os"
	"pangolin/pkg/cli"
	"pangolin/pkg/parser"
	"pangolin/pkg/path"
	"strings"

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
 
type TUI struct {
	program    *tea.Program
	mode       mode
	loginError string
	qrContent  string
 	checkingSession bool
	jbox       cli.JboxClient

	info     []string
	lines    []string
	input    textinput.Model
	viewport viewport.Model
	width    int
	height   int
	pathMgr  path.PathManager
	parser   parser.Parser
}

func NewTUI(pathMgr path.PathManager, parser parser.Parser, jbox cli.JboxClient) tea.Model {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Prompt = ""
	ti.Focus()

	vp := viewport.New(0, 0)

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
		input:    ti,
		viewport: vp,
		parser:   parser,
		pathMgr:  pathMgr,
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

func (t *TUI) exec(inputLine string) {
	inputLine = strings.TrimSpace(inputLine)
	if inputLine == "" {
		return
	}

	t.lines = append(t.lines, t.promptStr()+inputLine)

	if inputLine == "clear" {
		t.lines = []string{}
		t.sync()
		t.viewport.GotoBottom()
		return
	}

	command := t.parser.Parse(inputLine)
	if command == nil {
		return
	}

	var outputBuffer bytes.Buffer
	err := command.Execute(os.Stdin, &outputBuffer)
	if err != nil {
		t.lines = append(t.lines, "Error: "+err.Error())
	}

	outputStr := outputBuffer.String()
	if outputStr != "" {
		outputStr = strings.ReplaceAll(outputStr, "\r", "")
		outputLines := strings.Split(strings.TrimRight(outputStr, "\n"), "\n")
		for _, line := range outputLines {
			if line != "" {
				t.lines = append(t.lines, line)
			}
		}
	}

	t.sync()
	t.viewport.GotoBottom()
}

func (t *TUI) sync() {
	var sb strings.Builder
	sb.WriteString(titleStyle.Render("SHELL"))
	sb.WriteString("\n\n")
	for _, l := range t.lines {
		sb.WriteString(l)
		sb.WriteString("\n")
	}
	t.viewport.SetContent(sb.String())
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
		t.sync()
		return t, nil

	case msgLoginError:
		t.loginError = msg.err
 		t.checkingSession = false
		return t, nil

 	case msgCheckingSession:
 		t.checkingSession = true
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
		}

	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height

		promptLen := lipgloss.Width(t.promptStr())
		t.input.Width = msg.Width - promptLen - 6

		baseBox := boxStyle.Copy().Width(t.width - 2)
		info := strings.Join(t.info, "\n")
		header := baseBox.Render(titleStyle.Render("🦔 PANGOLIN SESSION INFO") + "\n\n" + info)
		headerHeight := lipgloss.Height(header)

		shellOuterHeight := t.height - headerHeight - 1 - 2
		viewportHeight := shellOuterHeight - 2

		if viewportHeight < 3 {
			viewportHeight = 3
		}

		t.viewport.Width = t.width - 4
		t.viewport.Height = viewportHeight

		t.sync()
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
	baseBox := boxStyle.Copy().Width(t.width - 2)
	info := strings.Join(t.info, "\n")
	header := baseBox.Render(titleStyle.Render("🦔 PANGOLIN SESSION INFO") + "\n\n" + info)

	var shellContainer strings.Builder
	shellContainer.WriteString(t.viewport.View())
	shellContainer.WriteString("\n\n")
	shellContainer.WriteString(t.promptStr())
	shellContainer.WriteString(t.input.View())

	shellOuterHeight := t.viewport.Height + 2
	shellBoxStyle := baseBox.Copy().Height(shellOuterHeight)
	shell := shellBoxStyle.Render(shellContainer.String())

	return lipgloss.JoinVertical(lipgloss.Left, header, "", shell)
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

	innerW := t.width - 12
	if innerW < 30 {
		innerW = 30
	}
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
