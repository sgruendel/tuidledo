package toodledo

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDoReturnsUnauthorizedError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"errorCode":2,"errorDesc":"Unauthorized"}`))
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	client := NewClient("", "", "")
	_, err = client.do(req)
	var unauthorized UnauthorizedError
	if !errors.As(err, &unauthorized) {
		t.Fatalf("error = %T %v, want UnauthorizedError", err, err)
	}
}

func TestAddTaskIncludesNote(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		payload := r.FormValue("tasks")
		var tasks []map[string]any
		if err := json.Unmarshal([]byte(payload), &tasks); err != nil {
			t.Fatal(err)
		}
		if len(tasks) != 1 {
			t.Fatalf("tasks len = %d, want 1", len(tasks))
		}
		if tasks[0]["note"] != "new note" {
			t.Fatalf("note payload = %#v, want new note", tasks[0]["note"])
		}
		_, _ = io.WriteString(w, `[{"id":1,"title":"new task","note":"new note","priority":1,"startdate":0,"context":0}]`)
	}))
	defer server.Close()

	client := NewClient("", "", "token")
	client.HTTPClient = server.Client()

	oldBaseURL := apiBaseURL
	apiBaseURL = server.URL
	defer func() { apiBaseURL = oldBaseURL }()

	_, err := client.AddTask(context.Background(), Task{Title: "new task", Note: "new note"})
	if err != nil {
		t.Fatal(err)
	}
}
