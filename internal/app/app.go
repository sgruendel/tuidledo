package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	datepicker "github.com/ethanefung/bubble-datepicker"

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
	stateEditTask
	stateConfirmDelete
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
	cfg    config.Config
	err    error
}

type deleteMsg struct {
	taskID int64
	cfg    config.Config
	err    error
}

type createMsg struct {
	task toodledo.Task
	cfg  config.Config
	err  error
}

type editMsg struct {
	task toodledo.Task
	cfg  config.Config
	err  error
}

type editField int

const (
	editFieldTitle editField = iota
	editFieldNote
	editFieldPriority
	editFieldStart
	editFieldDue
	editFieldContext
	editFieldCount
)

type listRow struct {
	priority int
	task     *toodledo.Task
}

type Model struct {
	clientID            string
	clientSecret        string
	cfg                 config.Config
	client              *toodledo.Client
	state               state
	previous            state
	message             string
	err                 error
	contexts            []toodledo.Context
	contextIndex        int
	tasks               []toodledo.Task
	visible             []toodledo.Task
	rows                []listRow
	cursor              int
	activePriority      int
	collapsedPriorities map[int]bool
	query               string
	editTaskID          int64
	editField           editField
	titleInput          textinput.Model
	noteInput           textarea.Model
	editPriority        int
	startPicker         datepicker.Model
	duePicker           datepicker.Model
	editContext         int64
	deleteTaskID        int64
	width               int
	height              int
}

func New(clientID, clientSecret string) Model {
	cfg, err := config.Load()
	if clientID == "" {
		clientID = os.Getenv("TOODLEDO_CLIENT_ID")
	}
	if clientSecret == "" {
		clientSecret = os.Getenv("TOODLEDO_CLIENT_SECRET")
	}
	m := Model{cfg: cfg, state: stateLoading, message: "Starting tuidledo...\n\nIf authorization is needed, open the URL printed below and return here after approving access."}
	if err != nil {
		m.state = stateError
		m.err = err
		return m
	}
	m.clientID = clientID
	m.clientSecret = clientSecret
	m.client = toodledo.NewClient(clientID, clientSecret, cfg.AccessToken)
	return m
}

func (m Model) Init() tea.Cmd {
	return m.startupCmd()
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
			m.client = toodledo.NewClient(m.clientID, m.clientSecret, m.cfg.AccessToken)
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
		m.applyConfig(msg.cfg)
		m.removeTask(msg.taskID)
		m.message = "Completed task"
		m.refreshVisible()
		return m, nil
	case deleteMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		m.applyConfig(msg.cfg)
		m.removeTask(msg.taskID)
		m.message = "Deleted task"
		m.refreshVisible()
		return m, nil
	case createMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		m.applyConfig(msg.cfg)
		m.tasks = append(m.tasks, msg.task)
		m.clearCreateForm()
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
	case editMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		m.applyConfig(msg.cfg)
		m.updateTask(msg.task)
		m.state = stateDetails
		m.message = "Updated task"
		m.clearEditForm()
		m.refreshVisible()
		for i, task := range m.visible {
			if task.ID == msg.task.ID {
				m.cursor = i
				break
			}
		}
		return m, nil
	case tea.PasteMsg:
		return m.handlePaste(msg)
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) View() tea.View {
	return tea.NewView(m.viewString())
}

func (m Model) viewString() string {
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
	if m.state == stateEditTask {
		return m.editView()
	}
	if m.state == stateConfirmDelete {
		return m.confirmDeleteView()
	}
	return m.taskView()
}

func (m Model) handlePaste(msg tea.PasteMsg) (tea.Model, tea.Cmd) {
	if m.state == stateSearch {
		m.query += msg.Content
		m.refreshVisible()
		return m, nil
	}
	if m.state == stateCreate {
		return m.updateFocusedCreateInput(msg)
	}
	if m.state == stateEditTask {
		return m.updateFocusedEditInput(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
			if text := msg.Key().Text; text != "" {
				m.query += text
				m.refreshVisible()
			}
			return m, nil
		}
	}
	if m.state == stateCreate {
		switch key {
		case "esc":
			m.message = ""
			m.clearCreateForm()
			m.state = stateTasks
			return m, nil
		case "ctrl+c":
			return m, m.quitCmd()
		case "tab":
			m.focusCreateField((m.editField + 1) % 2)
			return m, nil
		case "shift+tab":
			m.focusCreateField((m.editField + 1) % 2)
			return m, nil
		case "enter":
			if m.editField == editFieldNote {
				return m.updateFocusedCreateInput(msg)
			}
			title, note, err := m.createdTaskValues()
			if err != nil {
				m.message = err.Error()
				return m, nil
			}
			m.message = "Creating task..."
			return m, m.createCmd(title, note)
		case "ctrl+s":
			title, note, err := m.createdTaskValues()
			if err != nil {
				m.message = err.Error()
				return m, nil
			}
			m.message = "Creating task..."
			return m, m.createCmd(title, note)
		}
		return m.updateFocusedCreateInput(msg)
	}
	if m.state == stateEditTask {
		switch key {
		case "esc":
			m.clearEditForm()
			m.state = stateDetails
			return m, nil
		case "ctrl+c":
			return m, m.quitCmd()
		case "tab":
			m.focusEditField((m.editField + 1) % editFieldCount)
			return m, nil
		case "shift+tab":
			m.focusEditField((m.editField + editFieldCount - 1) % editFieldCount)
			return m, nil
		case "enter":
			if m.editField == editFieldPriority {
				m.nextEditPriority()
				return m, nil
			}
			if m.editField == editFieldStart || m.editField == editFieldDue {
				m.selectFocusedDate()
				return m, nil
			}
			if m.editField == editFieldContext {
				m.nextEditContext()
				return m, nil
			}
		case "[":
			if m.editField == editFieldPriority {
				m.prevEditPriority()
				return m, nil
			}
			if m.editField == editFieldContext {
				m.prevEditContext()
				return m, nil
			}
		case "]":
			if m.editField == editFieldPriority {
				m.nextEditPriority()
				return m, nil
			}
			if m.editField == editFieldContext {
				m.nextEditContext()
				return m, nil
			}
		case "ctrl+s":
			if _, err := m.editedTask(time.Now()); err != nil {
				m.message = err.Error()
				return m, nil
			}
			m.message = "Updating task..."
			return m, m.editCmd()
		case "h", "left", "l", "right", "j", "down", "k", "up", "H", "L", "x":
			if m.editField == editFieldStart || m.editField == editFieldDue {
				m.updateFocusedDatePicker(key)
				return m, nil
			}
		}
		return m.updateFocusedEditInput(msg)
	}
	if m.state == stateConfirmDelete {
		switch key {
		case "y", "Y":
			if m.deleteTaskID == 0 {
				m.state = stateTasks
				return m, nil
			}
			taskID := m.deleteTaskID
			m.deleteTaskID = 0
			m.state = stateTasks
			m.message = "Deleting task..."
			return m, m.deleteCmd(taskID)
		case "n", "N", "esc", "q":
			m.deleteTaskID = 0
			m.state = stateTasks
			m.message = "Delete cancelled"
			return m, nil
		case "ctrl+c":
			return m, m.quitCmd()
		}
		return m, nil
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
		if m.cursor < len(m.rows)-1 {
			m.cursor++
			m.updateActivePriorityFromCursor()
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			m.updateActivePriorityFromCursor()
		}
	case "g", "home":
		m.cursor = 0
		m.updateActivePriorityFromCursor()
	case "G", "end":
		if len(m.rows) > 0 {
			m.cursor = len(m.rows) - 1
			m.updateActivePriorityFromCursor()
		}
	case "tab", ".":
		m.nextPriority()
	case "shift+tab", ",":
		m.prevPriority()
	case "h", "left":
		m.setActivePriorityCollapsed(true)
	case "l", "right":
		m.setActivePriorityCollapsed(false)
	case "]":
		m.nextContext()
	case "[":
		m.prevContext()
	case "/":
		m.state = stateSearch
	case "n":
		m.startCreateForm()
		return m, m.titleInput.Focus()
	case "enter":
		if m.currentTask() != nil {
			m.state = stateDetails
		} else if len(m.rows) > 0 {
			m.toggleActivePriority()
		}
	case "e":
		if m.state == stateDetails {
			task := m.currentTask()
			if task == nil {
				return m, nil
			}
			m.startEditForm(*task)
			return m, m.titleInput.Focus()
		}
	case "d":
		if task := m.currentTask(); task != nil {
			m.message = "Completing task..."
			return m, m.completeCmd(task.ID)
		}
	case "D":
		if task := m.currentTask(); task != nil {
			m.deleteTaskID = task.ID
			m.state = stateConfirmDelete
		}
	}
	return m, nil
}

func (m Model) startupCmd() tea.Cmd {
	return func() tea.Msg {
		if m.clientID == "" || m.clientSecret == "" {
			return syncMsg{err: fmt.Errorf("set TOODLEDO_CLIENT_ID and TOODLEDO_CLIENT_SECRET, then register redirect URI http://127.0.0.1:8765/callback")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		client := toodledo.NewClient(m.clientID, m.clientSecret, m.cfg.AccessToken)
		cfg := m.cfg
		cfg, client, err := refreshTokenIfNeeded(ctx, cfg, client)
		if err != nil {
			return syncMsg{err: err}
		}
		if cfg.AccessToken == "" || time.Now().After(cfg.TokenExpiry) {
			result, err := toodledo.WaitForAuthCode(ctx, m.clientID)
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
		var contexts []toodledo.Context
		var tasks []toodledo.Task
		cfg, _, err := m.refreshAndRetry(ctx, func(client *toodledo.Client) error {
			var err error
			contexts, tasks, err = fetchAll(ctx, client)
			return err
		})
		return syncMsg{contexts: contexts, tasks: tasks, cfg: cfg, err: err}
	}
}

func (m Model) completeCmd(taskID int64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cfg, _, err := m.refreshAndRetry(ctx, func(client *toodledo.Client) error {
			return client.CompleteTask(ctx, taskID, time.Now())
		})
		return completeMsg{taskID: taskID, cfg: cfg, err: err}
	}
}

func (m Model) deleteCmd(taskID int64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cfg, _, err := m.refreshAndRetry(ctx, func(client *toodledo.Client) error {
			return client.DeleteTask(ctx, taskID)
		})
		return deleteMsg{taskID: taskID, cfg: cfg, err: err}
	}
}

func (m Model) createCmd(title, note string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		task := toodledo.Task{
			Title:     title,
			Note:      note,
			Priority:  1,
			StartDate: toodledo.NoonUnix(time.Now()),
			Context:   m.currentContextID(),
		}
		var created toodledo.Task
		cfg, _, err := m.refreshAndRetry(ctx, func(client *toodledo.Client) error {
			var err error
			created, err = client.AddTask(ctx, task)
			return err
		})
		return createMsg{task: created, cfg: cfg, err: err}
	}
}

func (m Model) createdTaskValues() (string, string, error) {
	title := strings.TrimSpace(m.titleInput.Value())
	if title == "" {
		return "", "", fmt.Errorf("task title cannot be empty")
	}
	return title, m.noteInput.Value(), nil
}

func (m Model) editCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		task, err := m.editedTask(time.Now())
		if err != nil {
			return editMsg{err: err}
		}

		var edited toodledo.Task
		cfg, _, err := m.refreshAndRetry(ctx, func(client *toodledo.Client) error {
			var err error
			edited, err = client.EditTask(ctx, task)
			return err
		})
		return editMsg{task: edited, cfg: cfg, err: err}
	}
}

func (m Model) editedTask(now time.Time) (toodledo.Task, error) {
	current := m.taskByID(m.editTaskID)
	if current == nil {
		return toodledo.Task{}, fmt.Errorf("task %d not found", m.editTaskID)
	}
	title := strings.TrimSpace(m.titleInput.Value())
	if title == "" {
		return toodledo.Task{}, fmt.Errorf("task title cannot be empty")
	}
	startDate := datePickerUnix(m.startPicker)
	dueDate := datePickerUnix(m.duePicker)

	task := *current
	task.Title = title
	task.Note = m.noteInput.Value()
	task.Priority = m.editPriority
	task.StartDate = startDate
	task.DueDate = dueDate
	task.Context = m.editContext
	return task, nil
}

func (m Model) refreshAndRetry(ctx context.Context, operation func(*toodledo.Client) error) (config.Config, *toodledo.Client, error) {
	cfg := m.cfg
	client := toodledo.NewClient(m.clientID, m.clientSecret, cfg.AccessToken)
	var err error
	cfg, client, err = refreshTokenIfNeeded(ctx, cfg, client)
	if err != nil {
		return cfg, client, err
	}
	if err := operation(client); err != nil {
		var unauthorized toodledo.UnauthorizedError
		if !errors.As(err, &unauthorized) || cfg.RefreshToken == "" {
			return cfg, client, err
		}

		token, refreshErr := client.RefreshToken(ctx, cfg.RefreshToken)
		if refreshErr != nil {
			return cfg, client, refreshErr
		}
		cfg.AccessToken = token.AccessToken
		cfg.RefreshToken = token.RefreshToken
		cfg.TokenExpiry = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
		if saveErr := config.Save(cfg); saveErr != nil {
			return cfg, client, saveErr
		}
		client.AccessToken = cfg.AccessToken
		if retryErr := operation(client); retryErr != nil {
			return cfg, client, retryErr
		}
	}
	return cfg, client, nil
}

func refreshTokenIfNeeded(ctx context.Context, cfg config.Config, client *toodledo.Client) (config.Config, *toodledo.Client, error) {
	if cfg.RefreshToken == "" || time.Now().Before(cfg.TokenExpiry.Add(-5*time.Minute)) {
		return cfg, client, nil
	}
	token, err := client.RefreshToken(ctx, cfg.RefreshToken)
	if err != nil {
		return cfg, client, err
	}
	cfg.AccessToken = token.AccessToken
	cfg.RefreshToken = token.RefreshToken
	cfg.TokenExpiry = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	if err := config.Save(cfg); err != nil {
		return cfg, client, err
	}
	client.AccessToken = cfg.AccessToken
	return cfg, client, nil
}

func (m *Model) applyConfig(cfg config.Config) {
	if cfg.AccessToken == "" {
		return
	}
	m.cfg = cfg
	m.client = toodledo.NewClient(m.clientID, m.clientSecret, cfg.AccessToken)
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
	previousActive := m.activePriority
	base := m.baseVisibleTasks()
	m.visible = make([]toodledo.Task, 0, len(base))
	m.rows = make([]listRow, 0, len(base)+4)
	lastPriority := -2
	for _, task := range base {
		if task.Priority != lastPriority {
			m.rows = append(m.rows, listRow{priority: task.Priority})
			lastPriority = task.Priority
		}
		if m.collapsedPriorities[task.Priority] {
			continue
		}
		taskCopy := task
		m.rows = append(m.rows, listRow{priority: task.Priority, task: &taskCopy})
		m.visible = append(m.visible, task)
	}
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	groups := m.priorityGroups()
	if len(groups) > 0 && !priorityIn(m.activePriority, groups) {
		m.activePriority = groups[0]
	}
	if len(groups) > 0 && priorityIn(previousActive, groups) {
		m.activePriority = previousActive
	}
}

func (m Model) baseVisibleTasks() []toodledo.Task {
	contextID := int64(0)
	if m.contextIndex > 0 && m.contextIndex-1 < len(m.contexts) {
		contextID = m.contexts[m.contextIndex-1].ID
	}
	return myn.VisibleTasks(m.tasks, contextID, m.query, time.Now())
}

func (m *Model) removeTask(taskID int64) {
	for i := range m.tasks {
		if m.tasks[i].ID == taskID {
			m.tasks = append(m.tasks[:i], m.tasks[i+1:]...)
			return
		}
	}
}

func (m *Model) updateTask(updated toodledo.Task) {
	for i := range m.tasks {
		if m.tasks[i].ID == updated.ID {
			m.tasks[i] = updated
			return
		}
	}
	m.tasks = append(m.tasks, updated)
}

func (m *Model) startEditForm(task toodledo.Task) {
	m.editTaskID = task.ID
	m.editContext = task.Context
	m.editPriority = task.Priority
	m.titleInput = newTitleInput(task.Title, m.width)
	m.noteInput = newNoteInput(task.Note, m.width)
	m.startPicker = newDatePicker(task.StartDate)
	m.duePicker = newDatePicker(task.DueDate)
	m.state = stateEditTask
	m.focusEditField(editFieldTitle)
}

func (m *Model) startCreateForm() {
	m.message = ""
	m.titleInput = newTitleInput("", m.width)
	m.noteInput = newNoteInput("", m.width)
	m.state = stateCreate
	m.focusCreateField(editFieldTitle)
}

func (m *Model) clearCreateForm() {
	m.editField = editFieldTitle
	m.titleInput = textinput.Model{}
	m.noteInput = textarea.Model{}
}

func (m *Model) clearEditForm() {
	m.editTaskID = 0
	m.editField = editFieldTitle
	m.editContext = 0
	m.editPriority = 0
	m.titleInput = textinput.Model{}
	m.noteInput = textarea.Model{}
	m.startPicker = datepicker.Model{}
	m.duePicker = datepicker.Model{}
}

func (m *Model) focusEditField(field editField) {
	m.editField = field
	m.titleInput.Blur()
	m.noteInput.Blur()
	m.startPicker.Blur()
	m.duePicker.Blur()
	switch field {
	case editFieldTitle:
		m.titleInput.Focus()
	case editFieldNote:
		m.noteInput.Focus()
	case editFieldStart:
		m.startPicker.SetFocus(datepicker.FocusCalendar)
	case editFieldDue:
		m.duePicker.SetFocus(datepicker.FocusCalendar)
	}
}

func (m *Model) focusCreateField(field editField) {
	m.editField = field
	m.titleInput.Blur()
	m.noteInput.Blur()
	if field == editFieldNote {
		m.noteInput.Focus()
		return
	}
	m.titleInput.Focus()
}

func (m Model) updateFocusedCreateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.editField == editFieldNote {
		m.noteInput, cmd = m.noteInput.Update(msg)
		return m, cmd
	}
	m.titleInput, cmd = m.titleInput.Update(msg)
	return m, cmd
}

func (m Model) updateFocusedEditInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.editField {
	case editFieldTitle:
		m.titleInput, cmd = m.titleInput.Update(msg)
	case editFieldNote:
		m.noteInput, cmd = m.noteInput.Update(msg)
	}
	return m, cmd
}

func (m *Model) nextEditContext() {
	ids := m.contextIDsForEdit()
	for i, id := range ids {
		if id == m.editContext {
			m.editContext = ids[(i+1)%len(ids)]
			return
		}
	}
	m.editContext = ids[0]
}

func (m *Model) prevEditContext() {
	ids := m.contextIDsForEdit()
	for i, id := range ids {
		if id == m.editContext {
			m.editContext = ids[(i+len(ids)-1)%len(ids)]
			return
		}
	}
	m.editContext = ids[0]
}

func (m *Model) nextEditPriority() {
	m.editPriority++
	if m.editPriority > 3 {
		m.editPriority = 0
	}
}

func (m *Model) prevEditPriority() {
	m.editPriority--
	if m.editPriority < 0 {
		m.editPriority = 3
	}
}

func (m *Model) selectFocusedDate() {
	if m.editField == editFieldStart {
		m.startPicker.SelectDate()
		return
	}
	if m.editField == editFieldDue {
		m.duePicker.SelectDate()
	}
}

func (m *Model) updateFocusedDatePicker(key string) {
	picker := &m.startPicker
	if m.editField == editFieldDue {
		picker = &m.duePicker
	}
	switch key {
	case "h", "left":
		picker.Yesterday()
		picker.SelectDate()
	case "l", "right":
		picker.Tomorrow()
		picker.SelectDate()
	case "j", "down":
		picker.NextWeek()
		picker.SelectDate()
	case "k", "up":
		picker.LastWeek()
		picker.SelectDate()
	case "H":
		picker.LastMonth()
		picker.SelectDate()
	case "L":
		picker.NextMonth()
		picker.SelectDate()
	case "x":
		picker.UnselectDate()
	}
}

func (m Model) contextIDsForEdit() []int64 {
	ids := make([]int64, 0, len(m.contexts)+1)
	ids = append(ids, 0)
	for _, contextItem := range m.contexts {
		ids = append(ids, contextItem.ID)
	}
	return ids
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
	priorities := m.priorityGroups()
	if len(priorities) == 0 {
		return
	}
	current := m.activePriority
	for i, priority := range priorities {
		if priority == current {
			m.setActivePriority(priorities[(i+1)%len(priorities)])
			return
		}
	}
	m.setActivePriority(priorities[0])
}

func (m *Model) prevPriority() {
	priorities := m.priorityGroups()
	if len(priorities) == 0 {
		return
	}
	current := m.activePriority
	for i, priority := range priorities {
		if priority == current {
			m.setActivePriority(priorities[(i+len(priorities)-1)%len(priorities)])
			return
		}
	}
	m.setActivePriority(priorities[len(priorities)-1])
}

func (m *Model) setActivePriority(priority int) {
	m.activePriority = priority
	for i, row := range m.rows {
		if row.priority == priority {
			m.cursor = i
			return
		}
	}
}

func (m *Model) updateActivePriorityFromCursor() {
	if len(m.rows) == 0 || m.cursor < 0 || m.cursor >= len(m.rows) {
		return
	}
	m.activePriority = m.rows[m.cursor].priority
}

func (m Model) currentTask() *toodledo.Task {
	if len(m.rows) == 0 || m.cursor < 0 || m.cursor >= len(m.rows) {
		return nil
	}
	return m.rows[m.cursor].task
}

func (m *Model) toggleActivePriority() {
	priorities := m.priorityGroups()
	if len(priorities) == 0 {
		return
	}
	if !priorityIn(m.activePriority, priorities) {
		m.activePriority = priorities[0]
	}
	if m.collapsedPriorities == nil {
		m.collapsedPriorities = make(map[int]bool)
	}
	m.collapsedPriorities[m.activePriority] = !m.collapsedPriorities[m.activePriority]
	m.refreshVisible()
	m.setActivePriority(m.activePriority)
}

func (m *Model) setActivePriorityCollapsed(collapsed bool) {
	priorities := m.priorityGroups()
	if len(priorities) == 0 {
		return
	}
	if !priorityIn(m.activePriority, priorities) {
		m.activePriority = priorities[0]
	}
	if m.collapsedPriorities == nil {
		m.collapsedPriorities = make(map[int]bool)
	}
	if m.collapsedPriorities[m.activePriority] == collapsed {
		return
	}
	m.collapsedPriorities[m.activePriority] = collapsed
	m.refreshVisible()
	m.setActivePriority(m.activePriority)
}

func (m Model) priorityGroups() []int {
	base := m.baseVisibleTasks()
	priorities := make([]int, 0, 4)
	seen := make(map[int]bool)
	for _, task := range base {
		if !seen[task.Priority] {
			seen[task.Priority] = true
			priorities = append(priorities, task.Priority)
		}
	}
	return priorities
}

func priorityIn(priority int, priorities []int) bool {
	for _, candidate := range priorities {
		if candidate == priority {
			return true
		}
	}
	return false
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

func (m Model) contextNameByID(contextID int64) string {
	if contextID == 0 {
		return "None"
	}
	for _, contextItem := range m.contexts {
		if contextItem.ID == contextID {
			return contextItem.Name
		}
	}
	return "Unknown"
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

	base := m.baseVisibleTasks()
	if len(base) == 0 {
		b.WriteString("No visible MYN tasks.\n")
	} else {
		lastPriority := -2
		row := 0
		for rowIndex, listRow := range m.rows {
			if listRow.task == nil {
				if rowIndex > 0 && listRow.priority != lastPriority {
					b.WriteByte('\n')
				}
				header := myn.PriorityLabel(listRow.priority)
				if m.collapsedPriorities[listRow.priority] {
					header += " (collapsed)"
				}
				if rowIndex == m.cursor {
					header = "> " + header
				} else {
					header = "  " + header
				}
				b.WriteString(priorityHeaderStyle.Render(fmt.Sprintf("%-48s  %-10s  %-10s  %-18s", header, "Start", "Due", "Repeat")))
				b.WriteByte('\n')
				lastPriority = listRow.priority
				continue
			}

			task := *listRow.task
			cursor := "  "
			style := taskRowStyle(row)
			if rowIndex == m.cursor {
				cursor = "> "
				style = selectedStyle
			}
			titleStyle := style
			if myn.IsToday(task.StartDate, time.Now()) {
				titleStyle = titleStyle.Underline(true)
			}
			b.WriteString(style.Render(cursor))
			b.WriteString(titleStyle.Render(fmt.Sprintf("%-46s", task.Title)))
			b.WriteString(style.Render(fmt.Sprintf("  %-10s  %-10s  %-18s", myn.DateLabel(task.StartDate), myn.DateLabel(task.DueDate), myn.RepeatLabel(task.Repeat))))
			b.WriteByte('\n')
			row++
		}
	}

	b.WriteString("\n")
	if m.message != "" {
		b.WriteString(subtleStyle.Render(m.message))
		b.WriteString("\n")
	}
	b.WriteString(helpStyle.Render("j/k move | n new | d done | D delete | h/l fold | ,/. priority | [ ] context | / search | enter details | r refresh | ? help | q quit"))
	b.WriteByte('\n')
	return b.String()
}

func (m Model) detailView() string {
	task := m.currentTask()
	if task == nil {
		return m.taskView()
	}
	return fmt.Sprintf("%s\n\n%s\n\nNote:\n%s\n\nPriority: %s\nStart: %s\nDue: %s\nRepeat: %s\nContext: %s\n\nAttachments:\n%s\n\n%s\n",
		titleStyle.Render("Task"), task.Title, linkURLs(emptyDash(task.Note)), myn.PriorityLabel(task.Priority), myn.DateLabel(task.StartDate), myn.DateLabel(task.DueDate), myn.RepeatLabel(task.Repeat), m.contextNameByID(task.Context), attachmentList(task.Attachment), helpStyle.Render("e edit | d complete | D delete | esc/q back"))
}

func (m Model) editView() string {
	priorityMarker := " "
	if m.editField == editFieldPriority {
		priorityMarker = ">"
	}
	contextMarker := " "
	if m.editField == editFieldContext {
		contextMarker = ">"
	}
	return fmt.Sprintf("%s\n\nTitle\n%s\n\nNote\n%s\n\nPriority\n%s %s\n\nStart\n%s\n\nDue\n%s\n\nContext\n%s %s\n\n%s\n",
		titleStyle.Render("Edit Task"),
		m.titleInput.View(),
		m.noteInput.View(),
		priorityMarker,
		myn.PriorityLabel(m.editPriority),
		m.dateFieldView(editFieldStart, m.startPicker),
		m.dateFieldView(editFieldDue, m.duePicker),
		contextMarker,
		m.contextNameByID(m.editContext),
		helpStyle.Render("tab next field | shift+tab previous | ctrl+s save | esc cancel | dates: h/j/k/l move, H/L month, enter select, x clear"))
}

func (m Model) dateFieldView(field editField, picker datepicker.Model) string {
	marker := " "
	if m.editField == field {
		marker = ">"
	}
	label := selectedDateText(picker)
	if m.editField != field {
		return fmt.Sprintf("%s %s", marker, label)
	}
	return fmt.Sprintf("%s %s\n%s", marker, label, picker.View())
}

func (m Model) createView() string {
	message := ""
	if m.message != "" {
		message = subtleStyle.Render(m.message) + "\n\n"
	}
	return fmt.Sprintf("%s\n\nContext: %s\nPriority: Med\nStart: %s\n\nTitle\n%s\n\nNote\n%s\n\n%s%s\n",
		titleStyle.Render("New Task"),
		m.contextName(),
		myn.DateLabel(toodledo.NoonUnix(time.Now())),
		m.titleInput.View(),
		m.noteInput.View(),
		message,
		helpStyle.Render("tab next field | shift+tab previous | enter create from title | ctrl+s create | esc cancel"))
}

func (m Model) confirmDeleteView() string {
	task := m.taskByID(m.deleteTaskID)
	title := "selected task"
	if task != nil {
		title = task.Title
	}
	return fmt.Sprintf("%s\n\nDelete this task permanently?\n\n%s\n\n%s\n",
		errorStyle.Render("Confirm Delete"), title, helpStyle.Render("y delete | n/esc cancel"))
}

func (m Model) taskByID(taskID int64) *toodledo.Task {
	for i := range m.tasks {
		if m.tasks[i].ID == taskID {
			return &m.tasks[i]
		}
	}
	return nil
}

func (m Model) helpView() string {
	return titleStyle.Render("Help") + `

j/k, arrows       move selection
g/G               jump to top/bottom
tab/shift+tab     jump between priority groups
.                 collapse/expand active priority group
[ / ]             switch context
/                 search visible task titles
n                 create new task in current context
d                 mark selected task done
D                 ask to delete selected task
e                 edit task from details
enter             show task details
r                 refresh from Toodledo
esc               back or clear search
q                 back, or quit from task list
ctrl+c            quit

Create form: tab/shift+tab switches fields, enter creates from title, ctrl+s creates, esc cancels.
Enter in the note field inserts a newline.

Edit form: tab/shift+tab switches fields, ctrl+s saves, esc cancels.
Priority and context fields cycle with [ ], or enter.
Date fields use h/j/k/l, H/L for months, enter to select, x to clear.

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

func newTitleInput(value string, width int) textinput.Model {
	input := textinput.New()
	input.Placeholder = "Task title"
	input.CharLimit = 255
	input.SetWidth(max(40, min(100, width-4)))
	input.SetValue(value)
	input.CursorEnd()
	return input
}

func newNoteInput(value string, width int) textarea.Model {
	input := textarea.New()
	input.Placeholder = "Task note"
	input.ShowLineNumbers = false
	input.SetWidth(max(40, min(100, width-4)))
	input.SetHeight(10)
	input.SetValue(value)
	return input
}

func newDatePicker(unix int64) datepicker.Model {
	date := time.Now()
	if unix != 0 {
		date = time.Unix(unix, 0).UTC()
	}
	picker := datepicker.New(date)
	picker.SetFocus(datepicker.FocusCalendar)
	if unix != 0 {
		picker.SelectDate()
	}
	return picker
}

func datePickerUnix(picker datepicker.Model) int64 {
	if !picker.Selected {
		return 0
	}
	return toodledo.NoonUnix(picker.Time)
}

func selectedDateText(picker datepicker.Model) string {
	if !picker.Selected {
		return "(unset)"
	}
	return myn.DateLabel(datePickerUnix(picker))
}

func linkURLs(value string) string {
	return urlPattern.ReplaceAllStringFunc(value, func(match string) string {
		trimmed := strings.TrimRight(match, ".,;:!?)]")
		trailing := strings.TrimPrefix(match, trimmed)
		return terminalLink(trimmed, trimmed) + trailing
	})
}

func terminalLink(url, label string) string {
	return "\x1b]8;;" + url + "\x1b\\" + label + "\x1b]8;;\x1b\\"
}

var urlPattern = regexp.MustCompile(`https?://[^\s<]+`)

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
