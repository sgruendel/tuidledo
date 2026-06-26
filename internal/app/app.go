package app

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sgruendel/tuidledo/internal/config"
	"github.com/sgruendel/tuidledo/internal/myn"
	"github.com/sgruendel/tuidledo/internal/toodledo"
)

type state int

const (
	stateLoading state = iota
	stateTasks
	stateDetails
	stateSearch
	stateCreate
	stateHelp
	stateError
)

type syncMsg struct {
	contexts []toodledo.Context
	tasks    []toodledo.Task
	cfg      config.Config
	err      error
}

type completeMsg struct {
	taskID int64
	err    error
}

type createMsg struct {
	task toodledo.Task
	err  error
}

type Model struct {
	cfg          config.Config
	client       *toodledo.Client
	state        state
	previous     state
	message      string
	err          error
	contexts     []toodledo.Context
	contextIndex int
	tasks        []toodledo.Task
	visible      []toodledo.Task
	cursor       int
	query        string
	createTitle  string
	width        int
	height       int
}

func New() Model {
	cfg, err := config.Load()
	if cfg.ClientID == "" {
		cfg.ClientID = os.Getenv("TOODLEDO_CLIENT_ID")
	}
	if cfg.ClientSecret == "" {
		cfg.ClientSecret = os.Getenv("TOODLEDO_CLIENT_SECRET")
	}
	m := Model{cfg: cfg, state: stateLoading, message: "Starting tuidledo...\n\nIf authorization is needed, open the URL printed below and return here after approving access."}
	if err != nil {
		m.state = stateError
		m.err = err
		return m
	}
	m.client = toodledo.NewClient(cfg.ClientID, cfg.ClientSecret, cfg.AccessToken)
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(tea.EnableBracketedPaste, m.startupCmd())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case syncMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		m.contexts = msg.contexts
		m.tasks = msg.tasks
		if msg.cfg.AccessToken != "" {
			m.cfg = msg.cfg
			m.client = toodledo.NewClient(m.cfg.ClientID, m.cfg.ClientSecret, m.cfg.AccessToken)
		}
		m.restoreContext()
		m.state = stateTasks
		m.message = fmt.Sprintf("Synced %d tasks", len(msg.tasks))
		m.refreshVisible()
		return m, nil
	case completeMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		for i := range m.tasks {
			if m.tasks[i].ID == msg.taskID {
				m.tasks = append(m.tasks[:i], m.tasks[i+1:]...)
				break
			}
		}
		m.message = "Completed task"
		m.refreshVisible()
		return m, nil
	case createMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		m.tasks = append(m.tasks, msg.task)
		m.createTitle = ""
		m.state = stateTasks
		m.message = "Created task"
		m.refreshVisible()
		for i, task := range m.visible {
			if task.ID == msg.task.ID {
				m.cursor = i
				break
			}
		}
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) View() string {
	if m.state == stateLoading {
		return titleStyle.Render("tuidledo") + "\n\n" + m.message + "\n"
	}
	if m.state == stateError {
		return titleStyle.Render("tuidledo") + "\n\n" + errorStyle.Render(m.err.Error()) + "\n\nq quit | r retry\n"
	}
	if m.state == stateHelp {
		return m.helpView()
	}
	if m.state == stateDetails {
		return m.detailView()
	}
	if m.state == stateCreate {
		return m.createView()
	}
	return m.taskView()
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if m.state == stateSearch {
		switch key {
		case "esc":
			m.query = ""
			m.state = stateTasks
			m.refreshVisible()
			return m, nil
		case "enter":
			m.state = stateTasks
			m.refreshVisible()
			return m, nil
		case "backspace", "ctrl+h":
			if len(m.query) > 0 {
				m.query = trimLastRune(m.query)
				m.refreshVisible()
			}
			return m, nil
		case "ctrl+c":
			return m, m.quitCmd()
		default:
			if msg.Type == tea.KeyRunes {
				m.query += string(msg.Runes)
				m.refreshVisible()
			}
			return m, nil
		}
	}
	if m.state == stateCreate {
		switch key {
		case "esc":
			m.createTitle = ""
			m.state = stateTasks
			return m, nil
		case "ctrl+c":
			return m, m.quitCmd()
		case "enter":
			title := strings.TrimSpace(m.createTitle)
			if title == "" {
				m.message = "Task title cannot be empty"
				m.state = stateTasks
				return m, nil
			}
			m.message = "Creating task..."
			return m, m.createCmd(title)
		case "backspace", "ctrl+h":
			if len(m.createTitle) > 0 {
				m.createTitle = trimLastRune(m.createTitle)
			}
			return m, nil
		default:
			if msg.Type == tea.KeyRunes {
				m.createTitle += string(msg.Runes)
			}
			return m, nil
		}
	}

	switch key {
	case "ctrl+c":
		return m, m.quitCmd()
	case "q":
		if m.state == stateTasks || m.state == stateError {
			return m, m.quitCmd()
		}
		m.state = stateTasks
		return m, nil
	case "esc":
		m.state = stateTasks
		if m.query != "" {
			m.query = ""
			m.refreshVisible()
		}
		return m, nil
	case "?":
		m.previous = m.state
		m.state = stateHelp
		return m, nil
	case "r":
		m.state = stateLoading
		m.message = "Refreshing tasks..."
		return m, m.syncCmd()
	case "j", "down":
		if m.cursor < len(m.visible)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		if len(m.visible) > 0 {
			m.cursor = len(m.visible) - 1
		}
	case "tab":
		m.nextPriority()
	case "shift+tab":
		m.prevPriority()
	case "]":
		m.nextContext()
	case "[":
		m.prevContext()
	case "/":
		m.state = stateSearch
	case "n":
		m.createTitle = ""
		m.state = stateCreate
	case "enter":
		if len(m.visible) > 0 {
			m.state = stateDetails
		}
	case " ":
		if len(m.visible) > 0 {
			task := m.visible[m.cursor]
			m.message = "Completing task..."
			return m, m.completeCmd(task.ID)
		}
	}
	return m, nil
}

func (m Model) startupCmd() tea.Cmd {
	return func() tea.Msg {
		if m.cfg.ClientID == "" || m.cfg.ClientSecret == "" {
			return syncMsg{err: fmt.Errorf("set TOODLEDO_CLIENT_ID and TOODLEDO_CLIENT_SECRET, then register redirect URI http://127.0.0.1:8765/callback")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		client := toodledo.NewClient(m.cfg.ClientID, m.cfg.ClientSecret, m.cfg.AccessToken)
		cfg := m.cfg
		if cfg.RefreshToken != "" && time.Now().After(cfg.TokenExpiry.Add(-5*time.Minute)) {
			token, err := client.RefreshToken(ctx, cfg.RefreshToken)
			if err == nil {
				cfg.AccessToken = token.AccessToken
				cfg.RefreshToken = token.RefreshToken
				cfg.TokenExpiry = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
				_ = config.Save(cfg)
				client.AccessToken = cfg.AccessToken
			}
		}
		if cfg.AccessToken == "" || time.Now().After(cfg.TokenExpiry) {
			result, err := toodledo.WaitForAuthCode(ctx, cfg.ClientID)
			if err != nil {
				return syncMsg{err: err}
			}
			token, err := client.ExchangeCode(ctx, result.Code)
			if err != nil {
				return syncMsg{err: err}
			}
			cfg.AccessToken = token.AccessToken
			cfg.RefreshToken = token.RefreshToken
			cfg.TokenExpiry = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
			if err := config.Save(cfg); err != nil {
				return syncMsg{err: err}
			}
			client.AccessToken = cfg.AccessToken
		}

		contexts, tasks, err := fetchAll(ctx, client)
		return syncMsg{contexts: contexts, tasks: tasks, cfg: cfg, err: err}
	}
}

func (m Model) syncCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		contexts, tasks, err := fetchAll(ctx, m.client)
		return syncMsg{contexts: contexts, tasks: tasks, err: err}
	}
}

func (m Model) completeCmd(taskID int64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return completeMsg{taskID: taskID, err: m.client.CompleteTask(ctx, taskID, time.Now())}
	}
}

func (m Model) createCmd(title string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		task := toodledo.Task{
			Title:     title,
			Priority:  1,
			StartDate: toodledo.NoonUnix(time.Now()),
			Context:   m.currentContextID(),
		}
		created, err := m.client.AddTask(ctx, task)
		return createMsg{task: created, err: err}
	}
}

func (m Model) quitCmd() tea.Cmd {
	m.cfg.LastContextID = m.currentContextID()
	return tea.Sequence(func() tea.Msg {
		_ = config.Save(m.cfg)
		return nil
	}, tea.Quit)
}

func fetchAll(ctx context.Context, client *toodledo.Client) ([]toodledo.Context, []toodledo.Task, error) {
	contexts, err := client.GetContexts(ctx)
	if err != nil {
		return nil, nil, err
	}
	tasks, err := client.GetTasks(ctx)
	if err != nil {
		return nil, nil, err
	}
	return contexts, tasks, nil
}

func (m *Model) refreshVisible() {
	contextID := int64(0)
	if m.contextIndex > 0 && m.contextIndex-1 < len(m.contexts) {
		contextID = m.contexts[m.contextIndex-1].ID
	}
	m.visible = myn.VisibleTasks(m.tasks, contextID, m.query, time.Now())
	if m.cursor >= len(m.visible) {
		m.cursor = len(m.visible) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *Model) nextContext() {
	m.contextIndex++
	if m.contextIndex > len(m.contexts) {
		m.contextIndex = 0
	}
	m.cursor = 0
	m.refreshVisible()
}

func (m *Model) prevContext() {
	m.contextIndex--
	if m.contextIndex < 0 {
		m.contextIndex = len(m.contexts)
	}
	m.cursor = 0
	m.refreshVisible()
}

func (m *Model) restoreContext() {
	m.contextIndex = 0
	if m.cfg.LastContextID == 0 {
		return
	}
	for i, contextItem := range m.contexts {
		if contextItem.ID == m.cfg.LastContextID {
			m.contextIndex = i + 1
			return
		}
	}
}

func (m *Model) nextPriority() {
	if len(m.visible) == 0 {
		return
	}
	current := m.visible[m.cursor].Priority
	for i := m.cursor + 1; i < len(m.visible); i++ {
		if m.visible[i].Priority != current {
			m.cursor = i
			return
		}
	}
	m.cursor = 0
}

func (m *Model) prevPriority() {
	if len(m.visible) == 0 {
		return
	}
	current := m.visible[m.cursor].Priority
	for i := m.cursor - 1; i >= 0; i-- {
		if m.visible[i].Priority != current {
			targetPriority := m.visible[i].Priority
			for i > 0 && m.visible[i-1].Priority == targetPriority {
				i--
			}
			m.cursor = i
			return
		}
	}
	lastPriority := m.visible[len(m.visible)-1].Priority
	for i := len(m.visible) - 1; i >= 0; i-- {
		if m.visible[i].Priority != lastPriority {
			m.cursor = i + 1
			return
		}
	}
	m.cursor = 0
}

func (m Model) contextName() string {
	if m.contextIndex == 0 {
		return "All"
	}
	if m.contextIndex-1 < len(m.contexts) {
		return m.contexts[m.contextIndex-1].Name
	}
	return "All"
}

func (m Model) currentContextID() int64 {
	if m.contextIndex > 0 && m.contextIndex-1 < len(m.contexts) {
		return m.contexts[m.contextIndex-1].ID
	}
	return 0
}

func (m Model) taskView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("tuidledo"))
	b.WriteString("  ")
	b.WriteString(subtleStyle.Render("context: " + m.contextName()))
	if m.query != "" || m.state == stateSearch {
		b.WriteString("  ")
		b.WriteString(subtleStyle.Render("/" + m.query))
	}
	b.WriteString("\n\n")

	if len(m.visible) == 0 {
		b.WriteString("No visible MYN tasks.\n")
	} else {
		lastPriority := -2
		row := 0
		for i, task := range m.visible {
			if task.Priority != lastPriority {
				if i > 0 {
					b.WriteByte('\n')
				}
				b.WriteString(priorityHeaderStyle.Render(fmt.Sprintf("  %-10s  %-10s  %-18s %s", "Start", "Due", "Repeat", myn.PriorityLabel(task.Priority))))
				b.WriteByte('\n')
				lastPriority = task.Priority
			}

			cursor := "  "
			style := taskRowStyle(row)
			if i == m.cursor {
				cursor = "> "
				style = selectedStyle
			}
			line := fmt.Sprintf("%s%-10s  %-10s  %-18s %s", cursor, myn.DateLabel(task.StartDate), myn.DateLabel(task.DueDate), myn.RepeatLabel(task.Repeat), task.Title)
			b.WriteString(style.Render(line))
			b.WriteByte('\n')
			row++
		}
	}

	b.WriteString("\n")
	if m.message != "" {
		b.WriteString(subtleStyle.Render(m.message))
		b.WriteString("\n")
	}
	b.WriteString(helpStyle.Render("j/k move | n new | tab/S-tab priority | [ ] context | / search | space complete | enter details | r refresh | ? help | q quit"))
	b.WriteByte('\n')
	return b.String()
}

func (m Model) detailView() string {
	if len(m.visible) == 0 {
		return m.taskView()
	}
	task := m.visible[m.cursor]
	return fmt.Sprintf("%s\n\n%s\n\nPriority: %s\nContext: %s\nStart: %s\nDue: %s\nRepeat: %s\n\nNote:\n%s\n\nAttachments:\n%s\n\n%s\n",
		titleStyle.Render("Task"), task.Title, myn.PriorityLabel(task.Priority), m.contextName(), myn.DateLabel(task.StartDate), myn.DateLabel(task.DueDate), myn.RepeatLabel(task.Repeat), emptyDash(task.Note), attachmentList(task.Attachment), helpStyle.Render("space complete | esc/q back"))
}

func (m Model) createView() string {
	return fmt.Sprintf("%s\n\nContext: %s\nPriority: Med\nStart: %s\n\nTask title:\n> %s\n\n%s\n",
		titleStyle.Render("New Task"), m.contextName(), myn.DateLabel(toodledo.NoonUnix(time.Now())), m.createTitle, helpStyle.Render("enter create | esc cancel"))
}

func (m Model) helpView() string {
	return titleStyle.Render("Help") + `

j/k, arrows       move selection
g/G               jump to top/bottom
tab/shift+tab     jump between priority groups
[ / ]             switch context
/                 search visible task titles
n                 create task in current context
space             complete selected task
enter             show task details
r                 refresh from Toodledo
esc               back or clear search
q                 back, or quit from task list
ctrl+c            quit

Register redirect URI: http://127.0.0.1:8765/callback
` + "\n"
}

func emptyDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func attachmentList(attachments []toodledo.Attachment) string {
	if len(attachments) == 0 {
		return "-"
	}
	var b strings.Builder
	for i, attachment := range attachments {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(emptyDash(attachment.Kind))
		b.WriteString(": ")
		b.WriteString(emptyDash(attachment.Name))
	}
	return b.String()
}

func trimLastRune(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return value
	}
	return string(runes[:len(runes)-1])
}

func taskRowStyle(row int) lipgloss.Style {
	if row%2 == 1 {
		return zebraStyle
	}
	return lipgloss.NewStyle()
}

var (
	titleStyle          = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	subtleStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	helpStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	errorStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	priorityHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("111"))
	zebraStyle          = lipgloss.NewStyle().Background(lipgloss.Color("235"))
	selectedStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("238"))
)
