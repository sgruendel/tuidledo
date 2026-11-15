package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/sgruendel/tuidledo/toodledo"
)

// datePicker is a month calendar for choosing a single date. Key handling
// lives in the model, which moves the date and toggles selection directly.
type datePicker struct {
	// Time is the highlighted date; its month is the one shown.
	Time time.Time
	// Selected reports whether Time is the chosen date or the field is unset.
	Selected bool
	// weekStart is the weekday shown in the first column.
	weekStart time.Weekday
	focused   bool
}

const datePickerCellWidth = 4

var weekdayHeaders = [...]string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}

func (p *datePicker) Focus() { p.focused = true }
func (p *datePicker) Blur()  { p.focused = false }

func (p *datePicker) SelectDate()   { p.Selected = true }
func (p *datePicker) UnselectDate() { p.Selected = false }

// Move shifts the highlighted date by the given number of months and days.
func (p *datePicker) Move(months, days int) {
	p.Time = p.Time.AddDate(0, months, days)
}

// View renders the month containing Time, with weeks starting on weekStart.
func (p datePicker) View() string {
	year, month, selectedDay := p.Time.Date()
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	daysInMonth := first.AddDate(0, 1, -1).Day()
	days := len(weekdayHeaders)
	leading := (int(first.Weekday()) - int(p.weekStart) + days) % days

	gridWidth := len(weekdayHeaders) * datePickerCellWidth
	var b strings.Builder
	b.WriteByte('\n')
	b.WriteString(lipgloss.PlaceHorizontal(gridWidth, lipgloss.Center,
		datePickerHeaderStyle.Render(fmt.Sprintf("%s %d", month, year))))
	b.WriteString("\n\n")

	for i := range days {
		header := weekdayHeaders[(int(p.weekStart)+i)%days]
		b.WriteString(datePickerHeaderStyle.Render(datePickerCell(header)))
	}

	for i := range leading + daysInMonth {
		if i%days == 0 {
			b.WriteString("\n\n")
		}
		if i < leading {
			b.WriteString(datePickerCell(""))
			continue
		}
		day := i - leading + 1
		style := datePickerDayStyle
		if p.Selected && day == selectedDay {
			style = datePickerSelectedStyle
			if p.focused {
				style = datePickerFocusedStyle
			}
		}
		b.WriteString(style.Render(datePickerCell(fmt.Sprintf("%02d", day))))
	}
	b.WriteByte('\n')
	return b.String()
}

func datePickerCell(text string) string {
	return fmt.Sprintf(" %-2s ", text)
}

func newDatePicker(unix int64, weekStart time.Weekday) datePicker {
	date := time.Now()
	if unix != 0 {
		date = time.Unix(unix, 0).UTC()
	}
	picker := datePicker{Time: date, Selected: unix != 0, weekStart: weekStart}
	picker.Focus()
	return picker
}

func datePickerUnix(picker datePicker) int64 {
	if !picker.Selected {
		return 0
	}
	return toodledo.NoonUnix(picker.Time)
}
