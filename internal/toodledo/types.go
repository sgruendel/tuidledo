package toodledo

import (
	"encoding/json"
	"time"
)

type Token struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token"`
}

type Context struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Task struct {
	ID         int64        `json:"id"`
	Title      string       `json:"title"`
	Modified   int64        `json:"modified"`
	Completed  int64        `json:"completed"`
	Priority   int          `json:"priority"`
	StartDate  int64        `json:"startdate"`
	DueDate    int64        `json:"duedate"`
	Repeat     string       `json:"repeat"`
	Context    int64        `json:"context"`
	Note       string       `json:"note"`
	Attachment []Attachment `json:"attachment"`
}

func (t *Task) UnmarshalJSON(data []byte) error {
	type taskAlias Task
	var raw struct {
		*taskAlias
		Attachment json.RawMessage `json:"attachment"`
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

type Attachment struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

type APIError struct {
	ErrorCode int    `json:"errorCode"`
	ErrorDesc string `json:"errorDesc"`
	Ref       string `json:"ref"`
}

func NoonUnix(t time.Time) int64 {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 12, 0, 0, 0, time.UTC).Unix()
}
