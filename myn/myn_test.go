package myn

import (
	"slices"
	"testing"
	"time"

	"github.com/sgruendel/tuidledo/toodledo"
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
	if ids := taskIDs(got); !slices.Equal(ids, wantIDs) {
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
	if ids := taskIDs(got); !slices.Equal(ids, wantIDs) {
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
	if ids := taskIDs(got); !slices.Equal(ids, wantIDs) {
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

func TestIsPastDate(t *testing.T) {
	now := time.Date(2026, 6, 22, 9, 0, 0, 0, time.UTC)
	if !IsPastDate(toodledo.NoonUnix(now.AddDate(0, 0, -1)), now) {
		t.Fatal("IsPastDate(yesterday) = false, want true")
	}
	if IsPastDate(toodledo.NoonUnix(now), now) {
		t.Fatal("IsPastDate(today) = true, want false")
	}
	if IsPastDate(toodledo.NoonUnix(now.AddDate(0, 0, 1)), now) {
		t.Fatal("IsPastDate(tomorrow) = true, want false")
	}
	if IsPastDate(0, now) {
		t.Fatal("IsPastDate(0) = true, want false")
	}
}

func taskIDs(tasks []toodledo.Task) []int64 {
	ids := make([]int64, 0, len(tasks))
	for _, task := range tasks {
		ids = append(ids, task.ID)
	}
	return ids
}

func TestDatesUseLocalCalendarDay(t *testing.T) {
	// In both cases the local date is a day off from the UTC date.
	tests := []struct {
		name string
		now  time.Time
	}{
		{"evening in New York", time.Date(2026, 11, 15, 20, 0, 0, 0, time.FixedZone("EST", -5*60*60))},
		{"after midnight in Berlin", time.Date(2026, 11, 15, 0, 30, 0, 0, time.FixedZone("CET", 1*60*60))},
	}
	day := func(d int) int64 { return time.Date(2026, 11, d, 12, 0, 0, 0, time.UTC).Unix() }

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for unix, want := range map[int64]string{day(14): "yesterday", day(15): "today", day(16): "tomorrow"} {
				if got := DateLabelAt(unix, test.now); got != want {
					t.Errorf("DateLabelAt(%v) = %q, want %q", time.Unix(unix, 0).UTC(), got, want)
				}
			}
			if !IsToday(day(15), test.now) || IsToday(day(16), test.now) {
				t.Error("IsToday does not match the local date")
			}
			if IsPastDate(day(15), test.now) || !IsPastDate(day(14), test.now) {
				t.Error("IsPastDate does not match the local date")
			}

			tasks := []toodledo.Task{
				{ID: 1, Title: "starts today", StartDate: day(15)},
				{ID: 2, Title: "starts tomorrow", StartDate: day(16)},
			}
			visible := VisibleTasks(tasks, 0, "", test.now)
			if len(visible) != 1 || visible[0].ID != 1 {
				t.Fatalf("VisibleTasks = %v, want only the task starting today", visible)
			}
		})
	}
}
