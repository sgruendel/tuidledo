package toodledo

import (
	"encoding/json"
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
