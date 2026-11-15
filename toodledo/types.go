package toodledo

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"time"
)

// Token is an OAuth token response from Toodledo.
type Token struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token"`
}

// Context is a Toodledo context.
type Context struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Priority is a Toodledo task priority.
//
// The values are defined by the Toodledo API and are sent and received as
// JSON numbers, so Priority deliberately has no text (un)marshaling methods.
type Priority int

// Toodledo task priorities.
const (
	PriorityNegative Priority = -1
	PriorityLow      Priority = 0
	PriorityMedium   Priority = 1
	PriorityHigh     Priority = 2
	PriorityTop      Priority = 3
)

// String returns the short display label for the priority.
func (p Priority) String() string {
	switch p {
	case PriorityTop:
		return "Top"
	case PriorityHigh:
		return "High"
	case PriorityMedium:
		return "Med"
	case PriorityLow:
		return "Low"
	default:
		return "Neg"
	}
}

// Next returns the next higher selectable priority, wrapping from Top to Low.
func (p Priority) Next() Priority {
	if p < PriorityLow || p >= PriorityTop {
		return PriorityLow
	}
	return p + 1
}

// Prev returns the next lower selectable priority, wrapping from Low to Top.
func (p Priority) Prev() Priority {
	if p <= PriorityLow || p > PriorityTop {
		return PriorityTop
	}
	return p - 1
}

// Task is a Toodledo task.
type Task struct {
	ID         int64        `json:"id"`
	Title      string       `json:"title"`
	Modified   int64        `json:"modified"`
	Completed  int64        `json:"completed"`
	Priority   Priority     `json:"priority"`
	StartDate  int64        `json:"startdate"`
	DueDate    int64        `json:"duedate"`
	Repeat     string       `json:"repeat"`
	Context    int64        `json:"context"`
	Note       string       `json:"note"`
	Attachment []Attachment `json:"attachment"`
}

// UnmarshalJSON decodes a task and normalizes Toodledo's attachment variants.
func (t *Task) UnmarshalJSON(data []byte) error {
	type taskAlias Task
	var raw struct {
		*taskAlias
		Attachment jsontext.Value `json:"attachment"`
	}
	raw.taskAlias = (*taskAlias)(t)
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if len(raw.Attachment) == 0 || string(raw.Attachment) == "null" || string(raw.Attachment) == "0" {
		t.Attachment = nil
		return nil
	}
	return json.Unmarshal(raw.Attachment, &t.Attachment)
}

// Attachment is a Toodledo task attachment.
type Attachment struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

// APIError is an error object returned by the Toodledo API.
type APIError struct {
	ErrorCode int    `json:"errorCode"`
	ErrorDesc string `json:"errorDesc"`
	Ref       string `json:"ref"`
}

// Error returns the error reported by Toodledo.
func (e APIError) Error() string {
	return fmt.Sprintf("toodledo API %d: %s", e.ErrorCode, e.ErrorDesc)
}

// NoonUnix returns the Unix timestamp for noon UTC on t's date.
func NoonUnix(t time.Time) int64 {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 12, 0, 0, 0, time.UTC).Unix()
}
