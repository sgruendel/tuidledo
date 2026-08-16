package myn

import (
	"cmp"
	"slices"
	"strings"
	"time"

	"github.com/sgruendel/tuidledo/toodledo"
)

// VisibleTasks filters and sorts tasks for the MYN task list.
func VisibleTasks(tasks []toodledo.Task, contextID int64, query string, now time.Time) []toodledo.Task {
	today := toodledo.NoonUnix(now)
	query = strings.ToLower(strings.TrimSpace(query))

	visible := make([]toodledo.Task, 0, len(tasks))
	for _, task := range tasks {
		if task.Completed != 0 || task.Priority < 0 {
			continue
		}
		if task.StartDate != 0 && task.StartDate > today {
			continue
		}
		if contextID != 0 && task.Context != contextID {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(task.Title), query) {
			continue
		}
		visible = append(visible, task)
	}

	slices.SortStableFunc(visible, func(left, right toodledo.Task) int {
		if left.Priority != right.Priority {
			return cmp.Compare(right.Priority, left.Priority)
		}
		if left.StartDate == 0 && right.StartDate != 0 {
			return 1
		}
		if left.StartDate != 0 && right.StartDate == 0 {
			return -1
		}
		if left.StartDate != right.StartDate {
			return cmp.Compare(right.StartDate, left.StartDate)
		}
		return cmp.Compare(strings.ToLower(left.Title), strings.ToLower(right.Title))
	})

	return visible
}

// PriorityLabel returns the short display label for a Toodledo priority.
func PriorityLabel(priority int) string {
	switch priority {
	case 3:
		return "Top"
	case 2:
		return "High"
	case 1:
		return "Med"
	case 0:
		return "Low"
	default:
		return "Neg"
	}
}

// DateLabel returns a relative display label for a Unix timestamp.
func DateLabel(unix int64) string {
	return DateLabelAt(unix, time.Now())
}

// DateLabelAt returns a relative display label for a Unix timestamp at now.
func DateLabelAt(unix int64, now time.Time) string {
	if unix == 0 {
		return "-"
	}
	date := dayStart(time.Unix(unix, 0).UTC())
	today := dayStart(now.UTC())
	switch date.Sub(today) {
	case -24 * time.Hour:
		return "yesterday"
	case 0:
		return "today"
	case 24 * time.Hour:
		return "tomorrow"
	default:
		return date.Format("2006-01-02")
	}
}

// IsToday reports whether a Unix timestamp falls on today's UTC date.
func IsToday(unix int64, now time.Time) bool {
	if unix == 0 {
		return false
	}
	return dayStart(time.Unix(unix, 0).UTC()).Equal(dayStart(now.UTC()))
}

// IsPastDate reports whether a Unix timestamp falls before today's UTC date.
func IsPastDate(unix int64, now time.Time) bool {
	if unix == 0 {
		return false
	}
	return dayStart(time.Unix(unix, 0).UTC()).Before(dayStart(now.UTC()))
}

func dayStart(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
