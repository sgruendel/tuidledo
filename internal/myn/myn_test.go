package myn

import (
	"reflect"
	"testing"
	"time"

	"github.com/sgruendel/tuidledo/internal/toodledo"
)

func TestVisibleTasksFiltersMYNTasks(t *testing.T) {
	now := time.Date(2026, 6, 22, 9, 0, 0, 0, time.UTC)
	tasks := []toodledo.Task{
		{ID: 1, Title: "visible", Priority: 1, StartDate: toodledo.NoonUnix(now)},
		{ID: 2, Title: "completed", Priority: 1, Completed: toodledo.NoonUnix(now)},
		{ID: 3, Title: "negative", Priority: -1},
		{ID: 4, Title: "future", Priority: 1, StartDate: toodledo.NoonUnix(now.AddDate(0, 0, 1))},
	}

	got := VisibleTasks(tasks, 0, "", now)
	wantIDs := []int64{1}
	if ids := taskIDs(got); !reflect.DeepEqual(ids, wantIDs) {
		t.Fatalf("VisibleTasks IDs = %v, want %v", ids, wantIDs)
	}
}

func TestVisibleTasksFiltersContextAndQuery(t *testing.T) {
	now := time.Date(2026, 6, 22, 9, 0, 0, 0, time.UTC)
	tasks := []toodledo.Task{
		{ID: 1, Title: "Write report", Priority: 1, Context: 10},
		{ID: 2, Title: "Buy milk", Priority: 1, Context: 20},
		{ID: 3, Title: "Report taxes", Priority: 1, Context: 20},
	}

	got := VisibleTasks(tasks, 20, "report", now)
	wantIDs := []int64{3}
	if ids := taskIDs(got); !reflect.DeepEqual(ids, wantIDs) {
		t.Fatalf("VisibleTasks IDs = %v, want %v", ids, wantIDs)
	}
}

func TestVisibleTasksSortsByPriorityThenStartDateDescending(t *testing.T) {
	now := time.Date(2026, 6, 22, 9, 0, 0, 0, time.UTC)
	tasks := []toodledo.Task{
		{ID: 1, Title: "old high", Priority: 2, StartDate: toodledo.NoonUnix(now.AddDate(0, 0, -10))},
		{ID: 2, Title: "medium", Priority: 1, StartDate: toodledo.NoonUnix(now)},
		{ID: 3, Title: "new high", Priority: 2, StartDate: toodledo.NoonUnix(now)},
		{ID: 4, Title: "no start high", Priority: 2},
	}

	got := VisibleTasks(tasks, 0, "", now)
	wantIDs := []int64{3, 1, 4, 2}
	if ids := taskIDs(got); !reflect.DeepEqual(ids, wantIDs) {
		t.Fatalf("VisibleTasks IDs = %v, want %v", ids, wantIDs)
	}
}

func TestDateLabelAtUsesRelativeLabels(t *testing.T) {
	now := time.Date(2026, 6, 22, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		date time.Time
		want string
	}{
		{name: "yesterday", date: now.AddDate(0, 0, -1), want: "yesterday"},
		{name: "today", date: now, want: "today"},
		{name: "tomorrow", date: now.AddDate(0, 0, 1), want: "tomorrow"},
		{name: "other", date: now.AddDate(0, 0, 2), want: "2026-06-24"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := DateLabelAt(toodledo.NoonUnix(test.date), now); got != test.want {
				t.Fatalf("DateLabelAt() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestIsToday(t *testing.T) {
	now := time.Date(2026, 6, 22, 9, 0, 0, 0, time.UTC)
	if !IsToday(toodledo.NoonUnix(now), now) {
		t.Fatal("IsToday() = false, want true")
	}
	if IsToday(toodledo.NoonUnix(now.AddDate(0, 0, -1)), now) {
		t.Fatal("IsToday(yesterday) = true, want false")
	}
	if IsToday(0, now) {
		t.Fatal("IsToday(0) = true, want false")
	}
}

func taskIDs(tasks []toodledo.Task) []int64 {
	ids := make([]int64, 0, len(tasks))
	for _, task := range tasks {
		ids = append(ids, task.ID)
	}
	return ids
}
