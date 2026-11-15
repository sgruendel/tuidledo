package toodledo

import (
	"encoding/json/v2"
	"testing"
)

func TestTaskUnmarshalAttachmentZero(t *testing.T) {
	var task Task
	if err := json.Unmarshal([]byte(`{"id":1,"title":"Task","attachment":0}`), &task); err != nil {
		t.Fatalf("unmarshal task: %v", err)
	}
	if len(task.Attachment) != 0 {
		t.Fatalf("Attachment len = %d, want 0", len(task.Attachment))
	}
}

func TestTaskUnmarshalAttachmentNull(t *testing.T) {
	var task Task
	if err := json.Unmarshal([]byte(`{"id":1,"title":"Task","attachment":null}`), &task); err != nil {
		t.Fatalf("unmarshal task: %v", err)
	}
	if len(task.Attachment) != 0 {
		t.Fatalf("Attachment len = %d, want 0", len(task.Attachment))
	}
}

func TestTaskUnmarshalAttachmentArrayWithStringID(t *testing.T) {
	var task Task
	data := []byte(`{"id":1,"title":"Task","note":"hello","attachment":[{"id":"abc","kind":"note","name":"Project notes"}]}`)
	if err := json.Unmarshal(data, &task); err != nil {
		t.Fatalf("unmarshal task: %v", err)
	}
	if task.Note != "hello" {
		t.Fatalf("Note = %q, want hello", task.Note)
	}
	if len(task.Attachment) != 1 {
		t.Fatalf("Attachment len = %d, want 1", len(task.Attachment))
	}
	if task.Attachment[0].Kind != "note" || task.Attachment[0].Name != "Project notes" {
		t.Fatalf("Attachment = %#v", task.Attachment[0])
	}
}

func TestPriorityString(t *testing.T) {
	tests := map[Priority]string{
		PriorityNegative: "Neg",
		PriorityLow:      "Low",
		PriorityMedium:   "Med",
		PriorityHigh:     "High",
		PriorityTop:      "Top",
	}
	for priority, want := range tests {
		if got := priority.String(); got != want {
			t.Fatalf("Priority(%d).String() = %q, want %q", int(priority), got, want)
		}
	}
}

func TestPriorityNextPrevWrap(t *testing.T) {
	order := []Priority{PriorityLow, PriorityMedium, PriorityHigh, PriorityTop}
	for i, priority := range order {
		if got, want := priority.Next(), order[(i+1)%len(order)]; got != want {
			t.Fatalf("%v.Next() = %v, want %v", priority, got, want)
		}
		if got, want := priority.Prev(), order[(i+len(order)-1)%len(order)]; got != want {
			t.Fatalf("%v.Prev() = %v, want %v", priority, got, want)
		}
	}
	if got := PriorityNegative.Next(); got != PriorityLow {
		t.Fatalf("Negative.Next() = %v, want Low", got)
	}
	if got := PriorityNegative.Prev(); got != PriorityTop {
		t.Fatalf("Negative.Prev() = %v, want Top", got)
	}
}

func TestPriorityJSONIsNumeric(t *testing.T) {
	var task Task
	if err := json.Unmarshal([]byte(`{"id":1,"priority":-1}`), &task); err != nil {
		t.Fatalf("unmarshal task: %v", err)
	}
	if task.Priority != PriorityNegative {
		t.Fatalf("Priority = %v, want Negative", task.Priority)
	}
	data, err := json.Marshal(map[string]any{"priority": PriorityTop})
	if err != nil {
		t.Fatalf("marshal priority: %v", err)
	}
	if string(data) != `{"priority":3}` {
		t.Fatalf("marshaled priority = %s, want {\"priority\":3}", data)
	}
}
