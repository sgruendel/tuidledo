package myn

import "testing"

func TestRepeatLabel(t *testing.T) {
	tests := []struct {
		name string
		rule string
		want string
	}{
		{name: "empty", rule: "", want: "-"},
		{name: "daily", rule: "FREQ=DAILY", want: "daily"},
		{name: "weekly interval", rule: "FREQ=WEEKLY;INTERVAL=2", want: "every 2 weeks"},
		{name: "weekly days", rule: "FREQ=WEEKLY;BYDAY=MO,WE,FR", want: "weekly on Mon, Wed, Fri"},
		{name: "monthly day", rule: "FREQ=MONTHLY;BYMONTHDAY=15", want: "monthly on day 15"},
		{name: "yearly month", rule: "FREQ=YEARLY;BYMONTH=5;BYMONTHDAY=10", want: "yearly on day 10 in May"},
		{name: "from completion", rule: "FREQ=DAILY;FROMCOMP", want: "daily from completion"},
		{name: "fast forward", rule: "FREQ=DAILY;FASTFORWARD", want: "daily fast-forward"},
		{name: "parent", rule: "PARENT", want: "with parent"},
		{name: "unknown", rule: "SOMETHING", want: "SOMETHING"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := RepeatLabel(test.rule); got != test.want {
				t.Fatalf("RepeatLabel(%q) = %q, want %q", test.rule, got, test.want)
			}
		})
	}
}
