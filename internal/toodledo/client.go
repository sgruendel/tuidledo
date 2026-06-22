package toodledo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const baseURL = "https://api.toodledo.com/3"

const taskFields = "priority,startdate,duedate,repeat,context,note,attachment"

type Client struct {
	HTTPClient   *http.Client
	AccessToken  string
	ClientID     string
	ClientSecret string
}

func NewClient(clientID, clientSecret, accessToken string) *Client {
	return &Client{
		HTTPClient:   &http.Client{Timeout: 20 * time.Second},
		AccessToken:  accessToken,
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}
}

func (c *Client) ExchangeCode(ctx context.Context, code string) (Token, error) {
	return c.token(ctx, url.Values{"grant_type": {"authorization_code"}, "code": {code}, "f": {"json"}})
}

func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (Token, error) {
	return c.token(ctx, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refreshToken}, "f": {"json"}})
}

func (c *Client) GetContexts(ctx context.Context) ([]Context, error) {
	var raw []json.RawMessage
	if err := c.get(ctx, "/contexts/get.php", nil, &raw); err != nil {
		return nil, err
	}
	contexts := make([]Context, 0, len(raw))
	for _, item := range raw {
		var apiErr APIError
		if json.Unmarshal(item, &apiErr) == nil && apiErr.ErrorCode != 0 {
			return nil, fmt.Errorf("toodledo contexts: %d %s", apiErr.ErrorCode, apiErr.ErrorDesc)
		}
		var ctxItem Context
		if err := json.Unmarshal(item, &ctxItem); err != nil {
			return nil, err
		}
		contexts = append(contexts, ctxItem)
	}
	return contexts, nil
}

func (c *Client) GetTasks(ctx context.Context) ([]Task, error) {
	params := url.Values{}
	params.Set("comp", "0")
	params.Set("fields", taskFields)

	var raw []json.RawMessage
	if err := c.get(ctx, "/tasks/get.php", params, &raw); err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, nil
	}

	tasks := make([]Task, 0, len(raw)-1)
	for i, item := range raw {
		if i == 0 {
			continue
		}
		var apiErr APIError
		if json.Unmarshal(item, &apiErr) == nil && apiErr.ErrorCode != 0 {
			return nil, fmt.Errorf("toodledo tasks: %d %s", apiErr.ErrorCode, apiErr.ErrorDesc)
		}
		var task Task
		if err := json.Unmarshal(item, &task); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (c *Client) AddTask(ctx context.Context, task Task) (Task, error) {
	payload, err := json.Marshal([]map[string]any{{
		"title":     task.Title,
		"priority":  task.Priority,
		"startdate": task.StartDate,
		"context":   task.Context,
	}})
	if err != nil {
		return Task{}, err
	}
	params := url.Values{}
	params.Set("tasks", string(payload))
	params.Set("fields", taskFields)

	var raw []json.RawMessage
	if err := c.post(ctx, "/tasks/add.php", params, &raw); err != nil {
		return Task{}, err
	}
	if len(raw) == 0 {
		return Task{}, fmt.Errorf("add task: empty response")
	}
	var apiErr APIError
	if json.Unmarshal(raw[0], &apiErr) == nil && apiErr.ErrorCode != 0 {
		return Task{}, fmt.Errorf("add task: %d %s", apiErr.ErrorCode, apiErr.ErrorDesc)
	}
	var added Task
	if err := json.Unmarshal(raw[0], &added); err != nil {
		return Task{}, err
	}
	return added, nil
}

func (c *Client) CompleteTask(ctx context.Context, taskID int64, completedAt time.Time) error {
	payload, err := json.Marshal([]map[string]any{{"id": taskID, "completed": NoonUnix(completedAt)}})
	if err != nil {
		return err
	}
	params := url.Values{}
	params.Set("tasks", string(payload))
	params.Set("reschedule", "1")
	params.Set("fields", taskFields)

	var raw []json.RawMessage
	if err := c.post(ctx, "/tasks/edit.php", params, &raw); err != nil {
		return err
	}
	for _, item := range raw {
		var apiErr APIError
		if json.Unmarshal(item, &apiErr) == nil && apiErr.ErrorCode != 0 {
			return fmt.Errorf("complete task: %d %s", apiErr.ErrorCode, apiErr.ErrorDesc)
		}
	}
	return nil
}

func (c *Client) token(ctx context.Context, params url.Values) (Token, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/account/token.php", strings.NewReader(params.Encode()))
	if err != nil {
		return Token{}, err
	}
	req.SetBasicAuth(c.ClientID, c.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	body, err := c.do(req)
	if err != nil {
		return Token{}, err
	}
	var apiErr APIError
	if json.Unmarshal(body, &apiErr) == nil && apiErr.ErrorCode != 0 {
		return Token{}, fmt.Errorf("token request: %d %s", apiErr.ErrorCode, apiErr.ErrorDesc)
	}
	var token Token
	if err := json.Unmarshal(body, &token); err != nil {
		return Token{}, err
	}
	return token, nil
}

func (c *Client) get(ctx context.Context, path string, params url.Values, dest any) error {
	if params == nil {
		params = url.Values{}
	}
	params.Set("access_token", c.AccessToken)
	params.Set("f", "json")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path+"?"+params.Encode(), nil)
	if err != nil {
		return err
	}
	body, err := c.do(req)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, dest)
}

func (c *Client) post(ctx context.Context, path string, params url.Values, dest any) error {
	params.Set("access_token", c.AccessToken)
	params.Set("f", "json")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewBufferString(params.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	body, err := c.do(req)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, dest)
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	hc := c.HTTPClient
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("toodledo http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}
