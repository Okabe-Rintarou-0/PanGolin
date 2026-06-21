package tui

import (
	"bufio"
	"bytes"
	"log"
	"os"
	"pangolin/pkg/cli"
	"pangolin/pkg/cmd"
	"pangolin/pkg/cmd/models"
	"pangolin/pkg/parser"
	"pangolin/pkg/path"
	"pangolin/pkg/tui/msg"
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

	history    []string
	historyIdx int

	executeChan  chan cmd.Command
	outputBuffer bytes.Buffer
}

func NewTUI(pathMgr path.PathManager, parser parser.Parser, jbox cli.JboxClient) *TUI {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Prompt = ""
	ti.Focus()

	vp := viewport.New(100, 20)
	executeChan := make(chan cmd.Command, 1)

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
		input:      ti,
		viewport:   vp,
		parser:     parser,
		pathMgr:    pathMgr,
		historyIdx: -1,

		executeChan: executeChan,
	}
}

func (t *TUI) Start(p *tea.Program) {
	t.program = p
	go t.executeThreadFn()
	f, _ := os.Create("tmp.log")
	log.SetOutput(f)
}

func (t *TUI) printErr(err error) {
	if err == nil {
		return
	}

	t.lines = append(t.lines, errorStyle.Render("Error: "+err.Error()))
}

func (t *TUI) executeThreadFn() {
	for cmd := range t.executeChan {
		if cmd == nil {
			continue
		}
		t.outputBuffer.Reset()
		err := cmd.Execute(os.Stdin, &t.outputBuffer)
		if err != nil {
			t.printErr(err)
		}

		scanner := bufio.NewScanner(&t.outputBuffer)
		for scanner.Scan() {
			line := scanner.Text()
			line = strings.ReplaceAll(line, "\r", "")
			if line != "" {
				t.lines = append(t.lines, line)
			}
		}

		t.program.Send(msg.ExecuteDoneMsg{})
	}
}

func (t *TUI) Init() tea.Cmd {
	go func() {
		if t.jbox.HasSession() {
			t.program.Send(msg.CheckingSessionMsg{})
		}

		err := t.jbox.Login(func(qrUrl string) {
			if t.program != nil {
				t.program.Send(msg.QRReadyMsg{Url: qrUrl})
			}
		})
		if err != nil {
			if t.program != nil {
				t.program.Send(msg.LoginErrorMsg{Err: err.Error()})
			}
		} else {
			if t.program != nil {
				t.program.Send(msg.LoginSuccessMsg{})
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

func (t *TUI) nextLineNo() int {
	return len(t.lines)
}

func (t *TUI) completePath() {
	inputVal := t.input.Value()
	if inputVal == "" || strings.Contains(inputVal, "|") {
		return
	}

	fields := parser.SplitArgs(inputVal)
	if len(fields) == 0 {
		return
	}
	cmdName := fields[0]
	cmdArgs := fields[1:]
	command := t.parser.ParseSingleCmd(t.nextLineNo(), cmdName, cmdArgs...)

	if command == nil {
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
	lastArgIdx := len(cmdArgs) - 1
	if lastArgIdx < 0 {
		lastArgIdx = 0
	}
	for i := 0; i < lastArgIdx; i++ {
		prefix.WriteString(" ")
		prefix.WriteString(cmdArgs[i])
	}
	prefix.WriteString(" ")

	currentPartial := ""
	if len(cmdArgs) > 0 {
		currentPartial = cmdArgs[len(cmdArgs)-1]
	}

	if len(hints) == 1 {
		val := hints[0].RealValue()
		if needsQuote(val) {
			val = addQuotes(val)
		}
		prefix.WriteString(val)
		t.input.SetValue(prefix.String())
		t.input.CursorEnd()
		return
	}

	commonPrefix := longestCommonPrefix(hints)
	if commonPrefix != currentPartial && commonPrefix != "" {
		val := commonPrefix
		if needsQuote(val) {
			val = addQuotes(val)
		}
		prefix.WriteString(val)
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

func needsQuote(val string) bool {
	return strings.ContainsAny(val, " ")
}

func addQuotes(val string) string {
	for _, p := range []string{"cloud:", "host:"} {
		if after, ok := strings.CutPrefix(val, p); ok {
			return p + "\"" + after + "\""
		}
	}
	return "\"" + val + "\""
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

func (t *TUI) clearShell() {
	t.lines = []string{}
	t.syncViewPort()
	t.viewport.GotoBottom()
}

func (t *TUI) executeAsync(inputLine string) {
	inputLine = strings.TrimSpace(inputLine)
	if inputLine == "" {
		return
	}

	if len(t.history) == 0 || t.history[len(t.history)-1] != inputLine {
		t.history = append(t.history, inputLine)
	}
	t.historyIdx = -1

	t.lines = append(t.lines, t.promptStr()+inputLine)
	t.syncViewPort()
	t.viewport.GotoBottom()

	if inputLine == "clear" {
		t.clearShell()
		return
	}
	command := t.parser.Parse(t.nextLineNo(), inputLine)
	if command == nil {
		return
	}
	t.executeChan <- command
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

func (t *TUI) showPrevHistoryCmd() {
	if t.mode == modeShell && len(t.history) > 0 {
		if t.historyIdx == -1 {
			t.historyIdx = len(t.history) - 1
		} else if t.historyIdx > 0 {
			t.historyIdx--
		}
		t.input.SetValue(t.history[t.historyIdx])
		t.input.CursorEnd()
	}
}

func (t *TUI) showNextHistoryCmd() {
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
}

func (t *TUI) Update(tmsg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch tmsg := tmsg.(type) {
	case msg.QRReadyMsg:
		t.qrContent = t.renderQR(tmsg.Url)
		t.checkingSession = false
		return t, nil

	case msg.ExecuteDoneMsg:
		t.syncViewPort()
		t.viewport.GotoBottom()

	case msg.LoginSuccessMsg:
		t.mode = modeShell
		t.info = t.jbox.SessionInfo()
		t.layoutViewport()
		t.syncViewPort()
		return t, nil

	case msg.LoginErrorMsg:
		t.loginError = tmsg.Err
		t.checkingSession = false
		return t, nil

	case msg.CheckingSessionMsg:
		t.checkingSession = true
		return t, nil

	case msg.ProgressMsg:
		lineIdx := tmsg.LineIdx
		total := tmsg.Total
		current := tmsg.Current
		err := tmsg.Err
		pbar := tmsg.Pbar
		if lineIdx >= len(t.lines) {
			needed := lineIdx + 1 - len(t.lines)
			t.lines = append(t.lines, make([]string, needed)...)
		}
		if total > 0 {
			pbar.Width = t.width - 4
			var bar string
			if err != nil {
				bar = errorStyle.Render("Error: " + err.Error())
			} else if current != total {
				pct := float64(current) / float64(total)
				bar = pbar.ViewAs(pct)
			} else {
				bar = "Done"
			}
			t.lines[lineIdx] = bar
			t.syncViewPort()
			t.viewport.GotoBottom()
		}
		return t, nil

	case tea.KeyMsg:
		switch tmsg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return t, tea.Quit
		case tea.KeyEnter:
			if t.mode == modeShell {
				t.executeAsync(t.input.Value())
				t.input.SetValue("")
			}
		case tea.KeyUp:
			t.showPrevHistoryCmd()
		case tea.KeyDown:
			t.showNextHistoryCmd()
		case tea.KeyTab:
			if t.mode == modeShell {
				t.completePath()
			}
		}

	case tea.WindowSizeMsg:
		t.width = tmsg.Width
		t.height = tmsg.Height
		t.layoutViewport()
		t.syncViewPort()
	}

	if t.mode == modeShell {
		t.input, cmd = t.input.Update(tmsg)
		cmds = append(cmds, cmd)

		t.viewport, cmd = t.viewport.Update(tmsg)
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
	sb.Grow(512)

	sb.WriteString("\n")
	sb.WriteString(titleStyle.Render("   🦔 PANGOLIN Login"))
	sb.WriteString("\n\n")

	if !t.checkingSession && t.loginError == "" {
		sb.WriteString(accentText.Render("  Scan the QR code with WeChat or SJTU App"))
		sb.WriteString("\n\n")
	}

	switch {
	case t.checkingSession:
		sb.WriteString("  ")
		sb.WriteString(mutedText.Render("📦 Checking session..."))
		sb.WriteString("\n\n")

	case t.loginError != "":
		sb.WriteString("  ")
		sb.WriteString(errorStyle.Render("✖ " + t.loginError)) // 错误信息动态拼接无法避免，但缩小了范围
		sb.WriteString("\n\n")

	case t.qrContent != "":
		sb.WriteString(t.qrContent)
		sb.WriteString("\n")

	default:
		sb.WriteString("  ")
		sb.WriteString(mutedText.Render("⟳ Loading QR code..."))
		sb.WriteString("\n\n")
	}

	sb.WriteString("\n  ")
	sb.WriteString(mutedText.Render("Ctrl+C to quit"))

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
