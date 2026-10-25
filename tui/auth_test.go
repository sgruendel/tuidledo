package tui

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	statepkg "github.com/sgruendel/tuidledo/state"
	"github.com/sgruendel/tuidledo/toodledo"
)

func TestInvalidRefreshTokenOffersSignIn(t *testing.T) {
	for _, expired := range []bool{true, false} {
		t.Run(map[bool]string{true: "startup refresh", false: "refresh after unauthorized"}[expired], func(t *testing.T) {
			requests := 0
			mockAPI(t, func(w http.ResponseWriter, r *http.Request) {
				requests++
				if r.URL.Path == "/3/account/token.php" {
					w.WriteHeader(http.StatusBadRequest)
					_, _ = io.WriteString(w, `{"errorCode":102,"errorDesc":"Invalid refresh token"}`)
					return
				}
				w.WriteHeader(http.StatusUnauthorized)
			})
			m := authTestModel(t)
			if expired {
				m.st.TokenExpiry = time.Now().Add(-time.Hour)
			}
			model, _ := m.Update(m.startupCmd()())
			m = model.(Model)
			if m.state != stateError || !errors.Is(m.err, toodledo.ErrInvalidRefreshToken) {
				t.Fatalf("startup state = %v, error = %v", m.state, m.err)
			}
			if !strings.Contains(m.viewString(), "r sign in again") {
				t.Fatalf("error view has no sign-in action: %s", m.viewString())
			}
			model, cmd := m.Update(keyPress("r"))
			m = model.(Model)
			if cmd == nil || m.state != stateLoading || m.message != "Starting sign-in..." {
				t.Fatalf("retry did not start sign-in: state = %v, message = %q", m.state, m.message)
			}
			// With missing app credentials, the command must fail as a sign-in
			// attempt rather than issue another request with the invalid token.
			m.clientID = ""
			m.state = stateError
			_, cmd = m.Update(keyPress("r"))
			if _, ok := cmd().(authStartedMsg); !ok {
				t.Fatal("retry did not return a sign-in command")
			}
			wantRequests := 1
			if !expired {
				wantRequests = 2
			}
			if requests != wantRequests {
				t.Fatalf("API requests = %d, want %d", requests, wantRequests)
			}
		})
	}
}

func TestReauthenticationDisplaysURLAndResumesSync(t *testing.T) {
	m := authTestModel(t)
	session := &toodledo.AuthSession{AuthURL: "https://example.test/authorize?state=random"}
	model, cmd := m.Update(authStartedMsg{session: session})
	m = model.(Model)
	t.Cleanup(m.authCancel)
	if cmd == nil || m.state != stateAuthorizing || !strings.Contains(m.viewString(), session.AuthURL) {
		t.Fatalf("sign-in did not display its URL: %s", m.viewString())
	}
	if _, duplicate := m.Update(keyPress("r")); duplicate != nil {
		t.Fatal("retry during sign-in started a second command")
	}
	st := statepkg.State{AccessToken: "new-access", RefreshToken: "new-refresh", TokenExpiry: time.Now().Add(time.Hour), LastContextID: 20}
	model, cmd = m.Update(authMsg{st: st})
	m = model.(Model)
	if m.state != stateLoading || cmd == nil || m.st != st || m.authSession != nil || m.authCancel != nil {
		t.Fatalf("sign-in did not resume sync: %+v", m)
	}
	mockAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("access_token") != "new-access" {
			t.Errorf("sync used stale token")
		}
		switch r.URL.Path {
		case "/3/contexts/get.php":
			_, _ = io.WriteString(w, `[{"id":20,"name":"Private"}]`)
		case "/3/tasks/get.php":
			_, _ = io.WriteString(w, `[{"num":0}]`)
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
		}
	})
	model, _ = m.Update(cmd())
	m = model.(Model)
	if m.state != stateTasks || m.currentContextID() != 20 {
		t.Fatalf("after sign-in state = %v, context = %d", m.state, m.currentContextID())
	}
}

func TestFailedSignInCanBeRetriedAndKeepsState(t *testing.T) {
	m := authTestModel(t)
	m.clientID = "" // Retrying should report missing setup, without opening a listener.
	original := m.st
	for _, msg := range []tea.Msg{
		authStartedMsg{err: errors.New("callback port unavailable")},
		authMsg{err: context.DeadlineExceeded},
		authMsg{err: errors.New("access_denied")},
		authMsg{err: toodledo.UnauthorizedError{Body: "Invalid client secret"}},
	} {
		model, _ := m.Update(msg)
		m = model.(Model)
		if m.state != stateError || requiresSignIn(m.err) || m.st != original {
			t.Fatalf("failed sign-in lost state or retry: state = %v, error = %v", m.state, m.err)
		}
		if !strings.Contains(m.viewString(), "r retry") || strings.Contains(m.viewString(), "Your Toodledo session needs authorization") {
			t.Fatalf("failed sign-in was described as an expired session: %s", m.viewString())
		}
		_, cmd := m.Update(keyPress("r"))
		if cmd == nil {
			t.Fatal("failed sign-in has no retry command")
		}
		if _, ok := cmd().(authStartedMsg); !ok {
			t.Fatal("failed sign-in retry did not restart the sign-in attempt")
		}
	}
}

func TestMissingCredentialsExplainSetup(t *testing.T) {
	m := authTestModel(t)
	m.clientSecret = ""
	model, _ := m.Update(m.startAuthCmd()())
	m = model.(Model)
	view := m.viewString()
	if requiresSignIn(m.err) || !strings.Contains(view, "TOODLEDO_CLIENT_SECRET") || !strings.Contains(view, "restart tuidledo") {
		t.Fatalf("missing credentials did not explain setup: %s", view)
	}
}

func TestRejectedCodeExchangeExplainsCredentials(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			m := authTestModel(t)
			original := m.st
			mockAPI(t, func(w http.ResponseWriter, r *http.Request) {
				if r.FormValue("grant_type") != "authorization_code" || r.FormValue("code") != "code" {
					t.Errorf("unexpected code exchange: %v", r.Form)
				}
				w.WriteHeader(status)
				_, _ = io.WriteString(w, `{"errorCode":102,"errorDesc":"Invalid client secret"}`)
			})
			msg := m.authorizeCmd(t.Context(), authCodeResult{result: toodledo.AuthResult{Code: "code"}})()
			model, _ := m.Update(msg)
			m = model.(Model)
			view := m.viewString()
			if requiresSignIn(m.err) || m.st != original || !strings.Contains(view, "check TOODLEDO_CLIENT_ID, TOODLEDO_CLIENT_SECRET") || !strings.Contains(view, "Invalid client secret") {
				t.Fatalf("rejected client credentials were not explained: %s", view)
			}
		})
	}
}

type authCodeResult struct {
	result toodledo.AuthResult
	err    error
}

func (r authCodeResult) Wait(context.Context) (toodledo.AuthResult, error) {
	return r.result, r.err
}

func TestTransientFailureRetriesWithoutSignIn(t *testing.T) {
	m := authTestModel(t)
	mockAPI(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	model, _ := m.Update(m.startupCmd()())
	m = model.(Model)
	if requiresSignIn(m.err) || !strings.Contains(m.viewString(), "r retry") {
		t.Fatalf("transient failure requested sign-in: %v", m.err)
	}
	model, cmd := m.Update(keyPress("r"))
	if model.(Model).message != "Retrying..." {
		t.Fatal("retry did not retain ordinary retry behavior")
	}
	if _, ok := cmd().(syncMsg); !ok {
		t.Fatal("transient retry started sign-in")
	}
}

func TestRefreshedTokensSurviveSubsequentFailure(t *testing.T) {
	m := authTestModel(t)
	m.st.TokenExpiry = time.Now().Add(-time.Hour)
	refreshes := 0
	mockAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/3/account/token.php" {
			refreshes++
			_, _ = io.WriteString(w, `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":7200}`)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	model, _ := m.Update(m.startupCmd()())
	m = model.(Model)
	if m.state != stateError || m.st.RefreshToken != "new-refresh" {
		t.Fatalf("refresh state was lost after sync failure: %+v", m.st)
	}
	_, cmd := m.Update(keyPress("r"))
	_ = cmd()
	if refreshes != 1 {
		t.Fatalf("refreshes = %d, want 1; retry reused a spent refresh token", refreshes)
	}
	_ = m.quitCmd()()
	saved, err := statepkg.Load()
	if err != nil || saved.RefreshToken != "new-refresh" || saved.LastContextID != 20 {
		t.Fatalf("quit saved stale credentials or context: %+v, %v", saved, err)
	}
}

func TestAuthenticationPathsPersistNewTokensAndKeepContext(t *testing.T) {
	for _, path := range []string{"authorization code", "expiring token", "unauthorized response"} {
		t.Run(path, func(t *testing.T) {
			m := authTestModel(t)
			if path == "expiring token" {
				m.st.TokenExpiry = time.Now().Add(-time.Hour)
			}
			tokenRequests := 0
			mockAPI(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/3/account/token.php":
					tokenRequests++
					wantGrant := "refresh_token"
					if path == "authorization code" {
						wantGrant = "authorization_code"
					}
					if r.FormValue("grant_type") != wantGrant {
						t.Errorf("grant_type = %q, want %q", r.FormValue("grant_type"), wantGrant)
					}
					_, _ = io.WriteString(w, `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":7200}`)
				default:
					if r.URL.Query().Get("access_token") != "new-access" {
						w.WriteHeader(http.StatusUnauthorized)
						return
					}
					_, _ = io.WriteString(w, `[]`)
				}
			})
			before := time.Now()
			cmd := m.startupCmd()
			if path == "authorization code" {
				cmd = m.authorizeCmd(t.Context(), authCodeResult{result: toodledo.AuthResult{Code: "code"}})
			}
			model, _ := m.Update(cmd())
			m = model.(Model)
			if m.state == stateError {
				t.Fatal(m.err)
			}
			saved, err := statepkg.Load()
			if err != nil {
				t.Fatal(err)
			}
			if tokenRequests != 1 || saved.AccessToken != "new-access" || saved.RefreshToken != "new-refresh" || saved.LastContextID != 20 {
				t.Fatalf("token requests = %d, persisted state = %+v", tokenRequests, saved)
			}
			if saved.TokenExpiry.Before(before.Add(2*time.Hour)) || saved.TokenExpiry.After(time.Now().Add(2*time.Hour)) {
				t.Fatalf("incorrect persisted token expiry: %v", saved.TokenExpiry)
			}
			if m.st.AccessToken != saved.AccessToken || m.st.RefreshToken != saved.RefreshToken || !m.st.TokenExpiry.Equal(saved.TokenExpiry) {
				t.Fatal("in-memory credentials differ from persisted credentials")
			}
		})
	}
}

func TestOperationErrorsKeepRefreshedTokens(t *testing.T) {
	st := statepkg.State{AccessToken: "new-access", RefreshToken: "new-refresh", TokenExpiry: time.Now().Add(time.Hour)}
	err := errors.New("request failed after token refresh")
	for _, msg := range []tea.Msg{
		syncMsg{st: st, err: err}, completeMsg{st: st, err: err}, deleteMsg{st: st, err: err},
		createMsg{st: st, err: err}, editMsg{st: st, err: err},
	} {
		m := Model{}
		model, _ := m.Update(msg)
		if updated := model.(Model); updated.st != st || updated.state != stateError {
			t.Fatalf("%T discarded refreshed tokens", msg)
		}
	}
}

func authTestModel(t *testing.T) Model {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_STATE_HOME", dir)
	t.Setenv("LOCALAPPDATA", dir)
	return Model{
		clientID: "client-id", clientSecret: "client-secret", state: stateLoading,
		st: statepkg.State{AccessToken: "old-access", RefreshToken: "old-refresh", TokenExpiry: time.Now().Add(time.Hour), LastContextID: 20},
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func mockAPI(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		handler(w, r)
		return w.Result(), nil
	})
}
