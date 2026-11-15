package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

func TestDatePickerViewLaysOutMonth(t *testing.T) {
	picker := datePicker{Time: time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)}
	view := ansi.Strip(picker.View())

	if !strings.Contains(view, "April 2026") {
		t.Fatalf("view missing month header:\n%s", view)
	}
	// April 2026 starts on a Wednesday, so three blank Sunday-first cells lead.
	if !strings.Contains(view, "\n"+strings.Repeat(" ", 3*datePickerCellWidth)+" 01  02  03  04 \n") {
		t.Fatalf("first week not aligned to Wednesday:\n%s", view)
	}
	if !strings.Contains(view, " 26  27  28  29  30 \n") {
		t.Fatalf("view missing last week:\n%s", view)
	}
}

func TestDatePickerViewHighlightsSelectedDay(t *testing.T) {
	picker := datePicker{Time: time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC), Selected: true}
	picker.Focus()
	if !strings.Contains(picker.View(), datePickerFocusedStyle.Render(datePickerCell("15"))) {
		t.Fatal("focused picker does not highlight the selected day")
	}
	picker.Blur()
	if !strings.Contains(picker.View(), datePickerSelectedStyle.Render(datePickerCell("15"))) {
		t.Fatal("blurred picker does not mark the selected day")
	}
}

func TestDatePickerMove(t *testing.T) {
	picker := datePicker{Time: time.Date(2026, 1, 31, 12, 0, 0, 0, time.UTC)}
	picker.Move(0, 7)
	if got := picker.Time.Format(time.DateOnly); got != "2026-02-07" {
		t.Fatalf("after +7 days = %s, want 2026-02-07", got)
	}
	picker.Move(-1, 0)
	if got := picker.Time.Format(time.DateOnly); got != "2026-01-07" {
		t.Fatalf("after -1 month = %s, want 2026-01-07", got)
	}
}
