package toodledo

import (
	"errors"
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
