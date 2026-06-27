package app

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/sgruendel/tuidledo/internal/toodledo"
)

func TestNavigationMovesCursor(t *testing.T) {
	m := testModel()

	m = updateKey(t, m, "j")
	if m.cursor != 1 {
		t.Fatalf("cursor after j = %d, want 1", m.cursor)
	}
	m = updateKey(t, m, "k")
	if m.cursor != 0 {
		t.Fatalf("cursor after k = %d, want 0", m.cursor)
	}
	m = updateKey(t, m, "G")
	if m.cursor != len(m.visible)-1 {
		t.Fatalf("cursor after G = %d, want %d", m.cursor, len(m.visible)-1)
	}
	m = updateKey(t, m, "g")
	if m.cursor != 0 {
		t.Fatalf("cursor after g = %d, want 0", m.cursor)
	}
}

func TestTabJumpsPriorityGroups(t *testing.T) {
	m := testModel()

	m = updateKey(t, m, "tab")
	if m.cursor != 2 {
		t.Fatalf("cursor after tab = %d, want first medium task at 2", m.cursor)
	}
	m = updateKey(t, m, "shift+tab")
	if m.cursor != 0 {
		t.Fatalf("cursor after shift+tab = %d, want first high task at 0", m.cursor)
	}
}

func TestBracketKeysSwitchContext(t *testing.T) {
	m := testModel()

	m = updateKey(t, m, "]")
	if m.contextIndex != 1 {
		t.Fatalf("contextIndex after ] = %d, want 1", m.contextIndex)
	}
	if len(m.visible) != 2 {
		t.Fatalf("visible after context switch = %d, want 2", len(m.visible))
	}
	m = updateKey(t, m, "[")
	if m.contextIndex != 0 {
		t.Fatalf("contextIndex after [ = %d, want 0", m.contextIndex)
	}
}

func TestSearchInputFiltersVisibleTasks(t *testing.T) {
	m := testModel()

	m = updateKey(t, m, "/")
	m = updateRunes(t, m, "third")
	if len(m.visible) != 1 || m.visible[0].Title != "third medium" {
		t.Fatalf("visible after search = %#v", m.visible)
	}
	m = updateKey(t, m, "esc")
	if m.query != "" {
		t.Fatalf("query after esc = %q, want empty", m.query)
	}
}

func TestCreateInputAcceptsPastedRunesAndCancels(t *testing.T) {
	m := testModel()

	m = updateKey(t, m, "n")
	m = updateRunes(t, m, "Pasted title")
	if m.createTitle != "Pasted title" {
		t.Fatalf("createTitle = %q, want pasted text", m.createTitle)
	}
	m = updateKey(t, m, "esc")
	if m.state != stateTasks || m.createTitle != "" {
		t.Fatalf("after esc state=%v createTitle=%q, want task state and empty title", m.state, m.createTitle)
	}
}

func TestEditFormInitializesFromDetails(t *testing.T) {
	m := testModel()
	m.tasks[0].Note = "old note"
	m.tasks[0].StartDate = toodledo.NoonUnix(time.Date(2026, 6, 22, 0, 0, 0, 0, time.UTC))
	m.tasks[0].DueDate = toodledo.NoonUnix(time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC))
	m.refreshVisible()

	m = updateKey(t, m, "enter")
	m = updateKey(t, m, "e")

	if m.state != stateEditTask {
		t.Fatalf("state after e = %v, want stateEditTask", m.state)
	}
	if m.editTaskID != 1 || m.titleInput.Value() != "first high" || m.noteInput.Value() != "old note" {
		t.Fatalf("edit form values: id=%d title=%q note=%q", m.editTaskID, m.titleInput.Value(), m.noteInput.Value())
	}
	if m.editPriority != 2 {
		t.Fatalf("editPriority = %d, want 2", m.editPriority)
	}
	if !m.startPicker.Selected || !m.duePicker.Selected {
		t.Fatalf("date pickers selected = start:%v due:%v, want both selected", m.startPicker.Selected, m.duePicker.Selected)
	}
	if got := datePickerUnix(m.startPicker); got != toodledo.NoonUnix(time.Date(2026, 6, 22, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("start date = %d", got)
	}
	if got := datePickerUnix(m.duePicker); got != toodledo.NoonUnix(time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("due date = %d", got)
	}
	if m.editContext != 10 {
		t.Fatalf("editContext = %d, want 10", m.editContext)
	}
}

func TestEditFormPrioritySelection(t *testing.T) {
	m := testModel()
	m = updateKey(t, m, "enter")
	m = updateKey(t, m, "e")
	m.focusEditField(editFieldPriority)

	m = updateKey(t, m, "]")
	if m.editPriority != 3 {
		t.Fatalf("editPriority after ] = %d, want 3", m.editPriority)
	}
	m = updateKey(t, m, "]")
	if m.editPriority != 0 {
		t.Fatalf("editPriority after wrap = %d, want 0", m.editPriority)
	}
	m = updateKey(t, m, "[")
	if m.editPriority != 3 {
		t.Fatalf("editPriority after [ = %d, want 3", m.editPriority)
	}
}

func TestEditFormTabSwitchesFields(t *testing.T) {
	m := testModel()
	m = updateKey(t, m, "enter")
	m = updateKey(t, m, "e")

	m = updateKey(t, m, "tab")
	if m.editField != editFieldNote {
		t.Fatalf("editField after tab = %v, want note", m.editField)
	}
	m = updateKey(t, m, "shift+tab")
	if m.editField != editFieldTitle {
		t.Fatalf("editField after shift+tab = %v, want title", m.editField)
	}
}

func TestEditFormContextSelection(t *testing.T) {
	m := testModel()
	m = updateKey(t, m, "enter")
	m = updateKey(t, m, "e")
	m.focusEditField(editFieldContext)

	m = updateKey(t, m, "]")
	if m.editContext != 20 {
		t.Fatalf("editContext after ] = %d, want 20", m.editContext)
	}
	m = updateKey(t, m, "[")
	if m.editContext != 10 {
		t.Fatalf("editContext after [ = %d, want 10", m.editContext)
	}
}

func TestEditFormDatePickerSelection(t *testing.T) {
	m := testModel()
	m = updateKey(t, m, "enter")
	m = updateKey(t, m, "e")
	m.focusEditField(editFieldStart)

	m = updateKey(t, m, "x")
	if m.startPicker.Selected {
		t.Fatal("startPicker.Selected after x = true, want false")
	}
	m = updateKey(t, m, "enter")
	if !m.startPicker.Selected {
		t.Fatal("startPicker.Selected after enter = false, want true")
	}
	before := m.startPicker.Time
	m = updateKey(t, m, "l")
	if !m.startPicker.Time.After(before) {
		t.Fatalf("startPicker did not move forward: before=%v after=%v", before, m.startPicker.Time)
	}
}

func TestEditMsgUpdatesTask(t *testing.T) {
	m := testModel()

	model, _ := m.Update(editMsg{task: toodledo.Task{ID: 1, Title: "updated", Priority: 3, Context: 20, Note: "new note"}})
	updated := model.(Model)
	if updated.tasks[0].Title != "updated" || updated.tasks[0].Note != "new note" || updated.tasks[0].Context != 20 || updated.tasks[0].Priority != 3 {
		t.Fatalf("updated task = %#v", updated.tasks[0])
	}
}

func TestDatePickerUnix(t *testing.T) {
	date := time.Date(2026, 6, 22, 0, 0, 0, 0, time.UTC)
	picker := newDatePicker(toodledo.NoonUnix(date))
	if got := datePickerUnix(picker); got != toodledo.NoonUnix(date) {
		t.Fatalf("datePickerUnix(selected) = %d", got)
	}
	picker.UnselectDate()
	if got := datePickerUnix(picker); got != 0 {
		t.Fatalf("datePickerUnix(unselected) = %d, want 0", got)
	}
}

func TestCompleteMsgRemovesTask(t *testing.T) {
	m := testModel()

	model, _ := m.Update(completeMsg{taskID: 1})
	updated := model.(Model)
	if len(updated.tasks) != 2 || updated.tasks[0].ID == 1 {
		t.Fatalf("tasks after complete = %#v", updated.tasks)
	}
	if updated.message != "Completed task" {
		t.Fatalf("message = %q, want Completed task", updated.message)
	}
}

func TestDeleteMsgRemovesTask(t *testing.T) {
	m := testModel()

	model, _ := m.Update(deleteMsg{taskID: 1})
	updated := model.(Model)
	if len(updated.tasks) != 2 || updated.tasks[0].ID == 1 {
		t.Fatalf("tasks after delete = %#v", updated.tasks)
	}
	if updated.message != "Deleted task" {
		t.Fatalf("message = %q, want Deleted task", updated.message)
	}
}

func TestDeleteKeyOpensConfirmation(t *testing.T) {
	m := testModel()

	m = updateKey(t, m, "D")
	if m.state != stateConfirmDelete {
		t.Fatalf("state after D = %v, want stateConfirmDelete", m.state)
	}
	if m.deleteTaskID != 1 {
		t.Fatalf("deleteTaskID = %d, want 1", m.deleteTaskID)
	}
}

func TestDeleteConfirmationCancel(t *testing.T) {
	m := testModel()
	m = updateKey(t, m, "D")
	m = updateKey(t, m, "n")

	if m.state != stateTasks {
		t.Fatalf("state after n = %v, want stateTasks", m.state)
	}
	if m.deleteTaskID != 0 {
		t.Fatalf("deleteTaskID = %d, want 0", m.deleteTaskID)
	}
	if len(m.tasks) != 3 {
		t.Fatalf("tasks after cancel = %d, want 3", len(m.tasks))
	}
}

func TestDeleteConfirmationConfirmReturnsCommand(t *testing.T) {
	m := testModel()
	m = updateKey(t, m, "D")

	msg := keyPress("y")
	model, cmd := m.Update(msg)
	updated := model.(Model)
	if updated.state != stateTasks {
		t.Fatalf("state after y = %v, want stateTasks", updated.state)
	}
	if updated.deleteTaskID != 0 {
		t.Fatalf("deleteTaskID = %d, want 0", updated.deleteTaskID)
	}
	if cmd == nil {
		t.Fatal("cmd after y = nil, want delete command")
	}
}

func TestLinkURLs(t *testing.T) {
	t.Setenv("TMUX", "")
	got := linkURLs("See https://example.com/test.")
	want := "See \x1b]8;;https://example.com/test\x1b\\https://example.com/test\x1b]8;;\x1b\\."
	if got != want {
		t.Fatalf("linkURLs() = %q, want %q", got, want)
	}
}

func testModel() Model {
	m := Model{
		state: stateTasks,
		contexts: []toodledo.Context{
			{ID: 10, Name: "Work"},
			{ID: 20, Name: "Private"},
		},
		tasks: []toodledo.Task{
			{ID: 1, Title: "first high", Priority: 2, Context: 10},
			{ID: 2, Title: "second high", Priority: 2, Context: 10},
			{ID: 3, Title: "third medium", Priority: 1, Context: 20},
		},
	}
	m.refreshVisible()
	return m
}

func updateKey(t *testing.T, m Model, key string) Model {
	t.Helper()
	model, _ := m.Update(keyPress(key))
	updatedModel, ok := model.(Model)
	if !ok {
		t.Fatalf("updated model type = %T, want app.Model", model)
	}
	return updatedModel
}

func updateRunes(t *testing.T, m Model, value string) Model {
	t.Helper()
	model, _ := m.Update(tea.PasteMsg{Content: value})
	updatedModel, ok := model.(Model)
	if !ok {
		t.Fatalf("updated model type = %T, want app.Model", model)
	}
	return updatedModel
}

func keyPress(key string) tea.KeyPressMsg {
	switch key {
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "shift+tab":
		return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEsc}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	default:
		runes := []rune(key)
		if len(runes) == 0 {
			return tea.KeyPressMsg{}
		}
		code := runes[0]
		if len(runes) == 1 && code >= 'A' && code <= 'Z' {
			return tea.KeyPressMsg{Code: code + ('a' - 'A'), ShiftedCode: code, Text: key, Mod: tea.ModShift}
		}
		return tea.KeyPressMsg{Code: code, Text: key}
	}
}
