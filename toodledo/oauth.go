package toodledo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

const callbackPath = "/callback"

// AuthResult contains the OAuth authorization callback details.
type AuthResult struct {
	Code        string
	State       string
	AuthURL     string
	RedirectURI string
}

// WaitForAuthCode starts a local callback server and waits for an OAuth code.
func WaitForAuthCode(ctx context.Context, clientID string) (AuthResult, error) {
	state, err := randomState()
	if err != nil {
		return AuthResult{}, err
	}

	listener, err := net.Listen("tcp", "127.0.0.1:8765")
	if err != nil {
		return AuthResult{}, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d%s", port, callbackPath)

	resultCh := make(chan AuthResult, 1)
	errCh := make(chan error, 1)
	server := &http.Server{ReadHeaderTimeout: 10 * time.Second}
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != callbackPath {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("state"); got != state {
			http.Error(w, "invalid oauth state", http.StatusBadRequest)
			errCh <- errors.New("invalid oauth state")
			return
		}
		if errText := r.URL.Query().Get("error"); errText != "" {
			http.Error(w, errText, http.StatusBadRequest)
			errCh <- errors.New(errText)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing oauth code", http.StatusBadRequest)
			errCh <- errors.New("missing oauth code")
			return
		}
		_, _ = w.Write([]byte("Authorization received. You can return to tuidledo."))
		resultCh <- AuthResult{Code: code, State: state, RedirectURI: redirectURI, AuthURL: authorizeURL(clientID, state)}
	})

	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	authURL := authorizeURL(clientID, state)
	fmt.Fprintf(os.Stderr, "\r\nOpen this URL to authorize tuidledo:\r\n\r\n%s\r\n\r\nRedirect URI: %s\r\n\r\nWaiting for browser callback...\r\n", authURL, redirectURI)
	select {
	case result := <-resultCh:
		result.AuthURL = authURL
		result.RedirectURI = redirectURI
		return result, nil
	case err := <-errCh:
		return AuthResult{}, err
	case <-ctx.Done():
		return AuthResult{}, ctx.Err()
	}
}

// authorizeURL constructs the Toodledo OAuth authorization URL.
func authorizeURL(clientID, state string) string {
	values := url.Values{}
	values.Set("response_type", "code")
	values.Set("client_id", clientID)
	values.Set("state", state)
	values.Set("scope", "basic tasks write")
	return baseURL + "/account/authorize.php?" + values.Encode()
}

// AuthURL returns the Toodledo OAuth authorization URL.
func AuthURL(clientID, state string) string {
	return authorizeURL(clientID, state)
}

// randomState generates a random state string for OAuth.
func randomState() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}
