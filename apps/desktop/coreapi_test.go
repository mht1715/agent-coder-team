package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDesktopAPIStoresTokensAndHidesThemFromFrontend(t *testing.T) {
	var meAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"username":"admin"},"tokens":{"access_token":"access-1","refresh_token":"refresh-1"}}`))
		case "/me":
			meAuthorization = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"username":"admin"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	api := NewDesktopAPI(nil, server.URL)
	login, err := api.Request(APIRequest{
		Method: http.MethodPost,
		Path:   "/auth/login",
		Body:   `{"username":"admin","password":"secret"}`,
	})
	if err != nil {
		t.Fatalf("login request: %v", err)
	}
	if login.Status != http.StatusOK {
		t.Fatalf("expected 200, got %d", login.Status)
	}
	if strings.Contains(login.Body, "access_token") || strings.Contains(login.Body, "refresh_token") {
		t.Fatalf("tokens leaked to frontend: %s", login.Body)
	}
	if !api.SessionStatus().Authenticated {
		t.Fatal("expected authenticated session")
	}

	me, err := api.Request(APIRequest{Method: http.MethodGet, Path: "/me"})
	if err != nil {
		t.Fatalf("me request: %v", err)
	}
	if me.Status != http.StatusOK {
		t.Fatalf("expected 200, got %d", me.Status)
	}
	if meAuthorization != "Bearer access-1" {
		t.Fatalf("unexpected authorization header: %q", meAuthorization)
	}
}

func TestDesktopAPIRefreshesExpiredSession(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{},"tokens":{"access_token":"expired","refresh_token":"refresh-1"}}`))
		case "/auth/refresh":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{},"tokens":{"access_token":"fresh","refresh_token":"refresh-2"}}`))
		case "/projects":
			callCount++
			if r.Header.Get("Authorization") == "Bearer expired" {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"code":"auth.unauthorized","message":"expired"}`))
				return
			}
			_, _ = w.Write([]byte(`{"items":[]}`))
		}
	}))
	defer server.Close()

	api := NewDesktopAPI(nil, server.URL)
	if _, err := api.Request(APIRequest{
		Method: http.MethodPost,
		Path:   "/auth/login",
		Body:   `{"username":"admin","password":"secret"}`,
	}); err != nil {
		t.Fatalf("login: %v", err)
	}

	response, err := api.Request(APIRequest{Method: http.MethodGet, Path: "/projects"})
	if err != nil {
		t.Fatalf("projects: %v", err)
	}
	if response.Status != http.StatusOK {
		t.Fatalf("expected 200 after refresh, got %d: %s", response.Status, response.Body)
	}
	if callCount != 2 {
		t.Fatalf("expected original request and retry, got %d calls", callCount)
	}
}

func TestDesktopAPIRejectsInvalidRequests(t *testing.T) {
	api := NewDesktopAPI(nil, "http://127.0.0.1:1")
	if _, err := api.Request(APIRequest{Method: "TRACE", Path: "/me"}); err == nil {
		t.Fatal("expected unsupported method to be rejected")
	}
	if _, err := api.Request(APIRequest{Method: http.MethodGet, Path: "https://example.com"}); err == nil {
		t.Fatal("expected absolute URL path to be rejected")
	}
	if _, err := api.Request(APIRequest{Method: http.MethodPost, Path: "/projects", Body: "{"}); err == nil {
		t.Fatal("expected invalid JSON body to be rejected")
	}
}

func TestSessionStatusJSON(t *testing.T) {
	payload, err := json.Marshal(SessionStatus{Authenticated: true})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(payload) != `{"authenticated":true}` {
		t.Fatalf("unexpected payload: %s", payload)
	}
}
