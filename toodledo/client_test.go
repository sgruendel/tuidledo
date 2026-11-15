package toodledo

import (
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDoReturnsUnauthorizedError(t *testing.T) {
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"errorCode":2,"errorDesc":"Unauthorized"}`))
	}))

	client := NewClient("", "", "")
	client.HTTPClient = server.Client()

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.do(req)
	if _, ok := errors.AsType[UnauthorizedError](err); !ok {
		t.Fatalf("error = %T %v, want UnauthorizedError", err, err)
	}
}

func TestRefreshTokenErrors(t *testing.T) {
	for _, tc := range []struct {
		name       string
		status     int
		body       string
		wantSignIn bool
	}{
		{"invalid refresh HTTP 400", http.StatusBadRequest, `{"errorCode":102,"errorDesc":"Invalid refresh token"}`, true},
		{"invalid refresh HTTP 200", http.StatusOK, `{"errorCode":102,"errorDesc":"Invalid refresh token"}`, true},
		{"invalid client", http.StatusBadRequest, `{"errorCode":102,"errorDesc":"Invalid client credentials"}`, false},
		{"rate limited", http.StatusTooManyRequests, `{"errorCode":103,"errorDesc":"Too many token requests"}`, false},
		{"server error", http.StatusInternalServerError, "server unavailable", false},
		{"incomplete token", http.StatusOK, `{"access_token":"new"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.FormValue("grant_type") != "refresh_token" || r.FormValue("refresh_token") != "old-refresh" {
					t.Errorf("unexpected token request: %v", r.Form)
				}
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			}))
			client := NewClient("client-id", "client-secret", "")
			client.HTTPClient = server.Client()
			_, err := client.RefreshToken(t.Context(), "old-refresh")
			if err == nil {
				t.Fatal("RefreshToken() error = nil")
			}
			if got := errors.Is(err, ErrInvalidRefreshToken); got != tc.wantSignIn {
				t.Fatalf("requires sign-in = %v, want %v; error = %v", got, tc.wantSignIn, err)
			}
		})
	}
}

func TestAPIResponseErrorsRemainInspectable(t *testing.T) {
	const errorBody = `{"errorCode":999,"errorDesc":"Request rejected","ref":"task-ref"}`
	for _, tc := range []struct {
		name string
		body string
		call func(*Client) error
	}{
		{"toodledo contexts", "[" + errorBody + "]", func(c *Client) error {
			_, err := c.Contexts(t.Context())
			return err
		}},
		{"toodledo tasks", "[" + errorBody + "]", func(c *Client) error {
			_, err := c.Tasks(t.Context())
			return err
		}},
		{"toodledo tasks", `[{"num":1},` + errorBody + "]", func(c *Client) error {
			_, err := c.Tasks(t.Context())
			return err
		}},
		{"add task", "[" + errorBody + "]", func(c *Client) error {
			_, err := c.AddTask(t.Context(), Task{Title: "task"})
			return err
		}},
		{"edit task", "[" + errorBody + "]", func(c *Client) error {
			_, err := c.EditTask(t.Context(), Task{ID: 1, Title: "task"})
			return err
		}},
		{"complete task", "[" + errorBody + "]", func(c *Client) error {
			return c.CompleteTask(t.Context(), 1, time.Now())
		}},
		{"delete task", "[" + errorBody + "]", func(c *Client) error {
			return c.DeleteTask(t.Context(), 1)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, tc.body)
			}))
			client := NewClient("", "", "token")
			client.HTTPClient = server.Client()
			err := tc.call(client)
			var apiErr APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("error = %v, want wrapped APIError", err)
			}
			if want := (APIError{ErrorCode: 999, ErrorDesc: "Request rejected", Ref: "task-ref"}); apiErr != want {
				t.Fatalf("APIError = %+v, want %+v", apiErr, want)
			}
			if !strings.HasPrefix(err.Error(), tc.name+": ") {
				t.Fatalf("error lost operation context: %v", err)
			}
		})
	}
}

func TestAddTaskSendsFormFields(t *testing.T) {
	startDate := NoonUnix(time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC))
	dueDate := NoonUnix(time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC))
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		want := map[string]any{
			"title":     "new task",
			"note":      "new note",
			"priority":  float64(PriorityHigh),
			"startdate": float64(startDate),
			"duedate":   float64(dueDate),
			"context":   float64(20),
		}
		for field, value := range want {
			if tasks[0][field] != value {
				t.Fatalf("%s payload = %#v, want %#v", field, tasks[0][field], value)
			}
		}
		_, _ = io.WriteString(w, `[{"id":1,"title":"new task","note":"new note","priority":2,"startdate":0,"context":20}]`)
	}))

	client := NewClient("", "", "token")
	client.HTTPClient = server.Client()

	oldBaseURL := apiBaseURL
	apiBaseURL = server.URL
	defer func() { apiBaseURL = oldBaseURL }()

	_, err := client.AddTask(t.Context(), Task{
		Title: "new task", Note: "new note", Priority: PriorityHigh,
		StartDate: startDate, DueDate: dueDate, Context: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
}
