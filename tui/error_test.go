package tui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestErrorEscapeRestoresFormOrTaskView(t *testing.T) {
	failure := errors.New("operation failed")
	for _, tc := range []struct {
		name   string
		origin state
		msg    tea.Msg
	}{
		{"sync", stateTasks, syncMsg{err: failure}},
		{"create", stateCreate, createMsg{err: failure}},
		{"edit", stateEditTask, editMsg{err: failure}},
		{"complete from list", stateTasks, completeMsg{err: failure}},
		{"complete from details", stateDetails, completeMsg{err: failure}},
		{"delete", stateTasks, deleteMsg{err: failure}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := selectFirstTask(t, testModel())
			switch tc.origin {
			case stateCreate:
				m.startCreateForm()
			case stateEditTask:
				m.startEditForm(*m.currentTask())
			default:
				m.state = tc.origin
			}
			m.titleInput = newTitleInput("unsaved title", m.width)
			m.noteInput = newNoteInput("unsaved note", m.width)
			model, _ := m.Update(tc.msg)
			m = model.(Model)
			if m.state != stateError || !strings.Contains(m.viewString(), "esc back | ? help") {
				t.Fatalf("operation error has no back/help navigation: %s", m.viewString())
			}
			model, cmd := m.Update(keyPress("esc"))
			m = model.(Model)
			if cmd != nil || m.state != tc.origin || m.err != nil {
				t.Fatalf("escape state = %v, want %v without a command", m.state, tc.origin)
			}
			if m.titleInput.Value() != "unsaved title" || m.noteInput.Value() != "unsaved note" || len(m.tasks) != 3 {
				t.Fatal("escape discarded the draft or changed the task list")
			}
		})
	}
}

func TestErrorHelpReturnsToErrorBeforeRestoringDraft(t *testing.T) {
	for _, backKey := range []string{"esc", "q", "?"} {
		t.Run(backKey, func(t *testing.T) {
			m := selectFirstTask(t, testModel())
			m.startEditForm(*m.currentTask())
			m.titleInput.SetValue("unsaved title")
			failure := errors.New("edit failed")
			model, _ := m.Update(editMsg{err: failure})
			m = updateKey(t, model.(Model), "?")
			if m.state != stateHelp {
				t.Fatal("? did not open help from the error screen")
			}
			m = updateKey(t, m, backKey)
			if m.state != stateError || !errors.Is(m.err, failure) {
				t.Fatal("closing help did not restore the error")
			}
			m = updateKey(t, m, "esc")
			if m.state != stateEditTask || m.titleInput.Value() != "unsaved title" {
				t.Fatal("help lost the route back to the draft")
			}
		})
	}
}

func TestEmptyEditTitleStaysEditable(t *testing.T) {
	m := selectFirstTask(t, testModel())
	m.startEditForm(*m.currentTask())
	m.titleInput.SetValue("")
	m.noteInput.SetValue("unsaved note")
	model, cmd := m.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	m = model.(Model)
	if cmd != nil || m.state != stateEditTask || m.message != "task title cannot be empty" || m.noteInput.Value() != "unsaved note" {
		t.Fatal("invalid title did not leave the edit form available for correction")
	}
}
