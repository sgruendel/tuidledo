package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

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

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}
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
	msg := tea.KeyMsg{Type: keyTypeForString(key), Runes: runesForString(key)}
	if key == "shift+tab" {
		msg.Type = tea.KeyShiftTab
	}
	model, _ := m.Update(msg)
	updatedModel, ok := model.(Model)
	if !ok {
		t.Fatalf("updated model type = %T, want app.Model", model)
	}
	return updatedModel
}

func updateRunes(t *testing.T, m Model, value string) Model {
	t.Helper()
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)})
	updatedModel, ok := model.(Model)
	if !ok {
		t.Fatalf("updated model type = %T, want app.Model", model)
	}
	return updatedModel
}

func keyTypeForString(key string) tea.KeyType {
	switch key {
	case "j", "k", "g", "G", "[", "]", "/", "n", "D", "y":
		return tea.KeyRunes
	case "tab":
		return tea.KeyTab
	case "esc":
		return tea.KeyEsc
	default:
		return tea.KeyRunes
	}
}

func runesForString(key string) []rune {
	switch key {
	case "tab", "shift+tab", "esc":
		return nil
	default:
		return []rune(key)
	}
}
