package toodledo

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestAuthSessionCallback(t *testing.T) {
	for _, tc := range []struct {
		name    string
		query   url.Values
		wantErr string
	}{
		{name: "success", query: url.Values{"code": {"authorization-code"}}},
		{name: "invalid state", query: url.Values{"state": {"wrong"}, "code": {"code"}}, wantErr: "invalid oauth state"},
		{name: "denied", query: url.Values{"error": {"access_denied"}}, wantErr: "access_denied"},
		{name: "missing code", query: url.Values{}, wantErr: "missing oauth code"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			session, err := startAuthCode("client-id", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(session.Close)
			authURL, err := url.Parse(session.AuthURL)
			if err != nil {
				t.Fatal(err)
			}
			if authURL.Query().Get("client_id") != "client-id" || authURL.Query().Get("state") == "" {
				t.Fatalf("invalid authorization URL: %s", session.AuthURL)
			}
			if !tc.query.Has("state") {
				tc.query.Set("state", authURL.Query().Get("state"))
			}
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Get(session.RedirectURI + "?" + tc.query.Encode())
			if err != nil {
				t.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			result, err := session.Wait(ctx)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("Wait() error = %v, want %s", err, tc.wantErr)
				}
			} else if err != nil || result.Code != "authorization-code" || result.RedirectURI != session.RedirectURI {
				t.Fatalf("Wait() = %+v, %v", result, err)
			}
			assertCallbackListenerReleased(t, session)
		})
	}
}

func TestAuthSessionCancellation(t *testing.T) {
	for _, closeSession := range []bool{false, true} {
		t.Run(map[bool]string{false: "context canceled", true: "session closed"}[closeSession], func(t *testing.T) {
			session, err := startAuthCode("client-id", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(session.Close)
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			if closeSession {
				session.Close()
			} else {
				cancel()
			}
			if _, err := session.Wait(ctx); !errors.Is(err, context.Canceled) {
				t.Fatalf("Wait() error = %v, want context.Canceled", err)
			}
			assertCallbackListenerReleased(t, session)
		})
	}
}

func TestStartAuthCodeExplainsOccupiedCallbackPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	address := listener.Addr().String()
	session, err := startAuthCode("client-id", address)
	if session != nil || err == nil || !strings.Contains(err.Error(), address) || !strings.Contains(err.Error(), "make sure this address is available") {
		t.Fatalf("occupied callback port did not explain setup: session = %v, error = %v", session, err)
	}
}

func assertCallbackListenerReleased(t *testing.T, session *AuthSession) {
	t.Helper()
	callbackURL, err := url.Parse(session.RedirectURI)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", callbackURL.Host)
	if err != nil {
		t.Fatalf("callback listener was not released: %v", err)
	}
	_ = listener.Close()
}
