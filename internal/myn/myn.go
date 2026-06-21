package myn

import (
	"sort"
	"strings"
	"time"

	"github.com/sgruendel/tuidledo/internal/toodledo"
)

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

	sort.SliceStable(visible, func(i, j int) bool {
		left, right := visible[i], visible[j]
		if left.Priority != right.Priority {
			return left.Priority > right.Priority
		}
		if left.DueDate == 0 && right.DueDate != 0 {
			return false
		}
		if left.DueDate != 0 && right.DueDate == 0 {
			return true
		}
		if left.DueDate != right.DueDate {
			return left.DueDate < right.DueDate
		}
		return strings.ToLower(left.Title) < strings.ToLower(right.Title)
	})

	return visible
}

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

func DateLabel(unix int64) string {
	if unix == 0 {
		return "-"
	}
	return time.Unix(unix, 0).UTC().Format("2006-01-02")
}
