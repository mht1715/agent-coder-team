package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const maxAPIResponseBytes = 10 << 20

type APIRequest struct {
	Method string            `json:"method"`
	Path   string            `json:"path"`
	Query  map[string]string `json:"query,omitempty"`
	Body   string            `json:"body,omitempty"`
}

type APIResponse struct {
	Status int    `json:"status"`
	Body   string `json:"body,omitempty"`
}

type SessionStatus struct {
	Authenticated bool `json:"authenticated"`
}

type ConversationStreamEvent struct {
	ConversationID string `json:"conversation_id"`
	Event          string `json:"event"`
	Data           string `json:"data,omitempty"`
}

type sessionTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type DesktopAPI struct {
	app          *application.App
	baseURL      string
	httpClient   *http.Client
	streamClient *http.Client

	tokensMu     sync.RWMutex
	accessToken  string
	refreshToken string
	refreshMu    sync.Mutex

	streamsMu sync.Mutex
	streams   map[string]context.CancelFunc
}

func NewDesktopAPI(app *application.App, baseURL string) *DesktopAPI {
	return &DesktopAPI{
		app:     app,
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		streamClient: &http.Client{},
		streams:      make(map[string]context.CancelFunc),
	}
}

func NewDesktopAPIFromEnvironment(app *application.App) *DesktopAPI {
	baseURL := os.Getenv("ACT_CORE_API_URL")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:18080/api/v1"
	}
	return NewDesktopAPI(app, baseURL)
}

func (a *DesktopAPI) Request(req APIRequest) (*APIResponse, error) {
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete:
	default:
		return nil, fmt.Errorf("unsupported method %q", req.Method)
	}
	path := strings.TrimSpace(req.Path)
	if !strings.HasPrefix(path, "/") || strings.Contains(path, "://") || strings.Contains(path, "..") {
		return nil, errors.New("invalid API path")
	}
	if req.Body != "" && !json.Valid([]byte(req.Body)) {
		return nil, errors.New("request body must be valid JSON")
	}

	status, body, err := a.execute(method, path, req.Query, req.Body, true)
	if err != nil {
		return nil, err
	}

	if status == http.StatusUnauthorized && a.hasRefreshToken() && !isPublicAuthPath(path) {
		if refreshed := a.refreshSession(); refreshed {
			status, body, err = a.execute(method, path, req.Query, req.Body, true)
			if err != nil {
				return nil, err
			}
		}
	}

	body = a.applySessionChanges(path, status, body)
	return &APIResponse{Status: status, Body: string(body)}, nil
}

func (a *DesktopAPI) SessionStatus() SessionStatus {
	a.tokensMu.RLock()
	defer a.tokensMu.RUnlock()
	return SessionStatus{Authenticated: a.accessToken != ""}
}

func (a *DesktopAPI) WatchConversation(conversationID string) error {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return errors.New("conversation id is required")
	}

	a.streamsMu.Lock()
	if cancel, ok := a.streams[conversationID]; ok {
		cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.streams[conversationID] = cancel
	a.streamsMu.Unlock()

	go a.consumeConversationStream(ctx, conversationID)
	return nil
}

func (a *DesktopAPI) UnwatchConversation(conversationID string) {
	a.streamsMu.Lock()
	cancel, ok := a.streams[conversationID]
	if ok {
		delete(a.streams, conversationID)
	}
	a.streamsMu.Unlock()
	if ok {
		cancel()
	}
}

func (a *DesktopAPI) ServiceShutdown() error {
	a.streamsMu.Lock()
	for id, cancel := range a.streams {
		cancel()
		delete(a.streams, id)
	}
	a.streamsMu.Unlock()
	return nil
}

func (a *DesktopAPI) execute(
	method, path string,
	query map[string]string,
	body string,
	withAuth bool,
) (int, []byte, error) {
	endpoint := a.baseURL + path
	if len(query) > 0 {
		values := url.Values{}
		for key, value := range query {
			if strings.TrimSpace(key) != "" && value != "" {
				values.Set(key, value)
			}
		}
		if encoded := values.Encode(); encoded != "" {
			endpoint += "?" + encoded
		}
	}

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	request, err := http.NewRequest(method, endpoint, reader)
	if err != nil {
		return 0, nil, fmt.Errorf("create request: %w", err)
	}
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	request.Header.Set("Accept", "application/json")
	if withAuth {
		if token := a.accessTokenValue(); token != "" {
			request.Header.Set("Authorization", "Bearer "+token)
		}
	}

	response, err := a.httpClient.Do(request)
	if err != nil {
		return 0, nil, fmt.Errorf("call core service: %w", err)
	}
	defer response.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(response.Body, maxAPIResponseBytes))
	if err != nil {
		return 0, nil, fmt.Errorf("read core response: %w", err)
	}
	return response.StatusCode, payload, nil
}

func (a *DesktopAPI) refreshSession() bool {
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()

	refreshToken := a.refreshTokenValue()
	if refreshToken == "" {
		return false
	}
	body, err := json.Marshal(map[string]string{"refresh_token": refreshToken})
	if err != nil {
		return false
	}
	status, payload, err := a.execute(http.MethodPost, "/auth/refresh", nil, string(body), false)
	if err != nil || status < 200 || status >= 300 {
		a.clearTokens()
		return false
	}
	a.applySessionChanges("/auth/refresh", status, payload)
	return a.accessTokenValue() != ""
}

func (a *DesktopAPI) applySessionChanges(path string, status int, body []byte) []byte {
	if status < 200 || status >= 300 {
		if path == "/auth/logout" {
			a.clearTokens()
		}
		return body
	}

	switch path {
	case "/auth/login", "/auth/register", "/auth/refresh":
		var envelope struct {
			Tokens *sessionTokens `json:"tokens"`
		}
		if err := json.Unmarshal(body, &envelope); err != nil || envelope.Tokens == nil {
			return body
		}
		a.setTokens(*envelope.Tokens)
		return withoutTokens(body)
	case "/auth/logout":
		a.clearTokens()
	}
	return body
}

func withoutTokens(body []byte) []byte {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return body
	}
	delete(payload, "tokens")
	sanitized, err := json.Marshal(payload)
	if err != nil {
		return body
	}
	return sanitized
}

func (a *DesktopAPI) consumeConversationStream(ctx context.Context, conversationID string) {
	for {
		if ctx.Err() != nil {
			return
		}
		err := a.streamConversationOnce(ctx, conversationID)
		if ctx.Err() != nil {
			return
		}
		a.emitStreamEvent(conversationID, "stream.error", errorData(err))
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func (a *DesktopAPI) streamConversationOnce(ctx context.Context, conversationID string) error {
	endpoint := a.baseURL + "/conversations/" + url.PathEscape(conversationID) + "/stream"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "text/event-stream")
	if token := a.accessTokenValue(); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	response, err := a.streamClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized && a.hasRefreshToken() && a.refreshSession() {
		return errors.New("session refreshed; reconnecting stream")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return fmt.Errorf("stream returned status %d", response.StatusCode)
	}

	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)
	eventName := ""
	dataLines := make([]string, 0, 1)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if eventName != "" && len(dataLines) > 0 {
				a.emitStreamEvent(conversationID, eventName, strings.Join(dataLines, "\n"))
			}
			eventName = ""
			dataLines = dataLines[:0]
			continue
		}
		switch {
		case strings.HasPrefix(line, "event:"):
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return errors.New("stream closed")
}

func (a *DesktopAPI) emitStreamEvent(conversationID, eventName, data string) {
	if a.app == nil {
		return
	}
	a.app.Event.Emit("conversation.event", ConversationStreamEvent{
		ConversationID: conversationID,
		Event:          eventName,
		Data:           data,
	})
}

func errorData(err error) string {
	if err == nil {
		return "{}"
	}
	payload, marshalErr := json.Marshal(map[string]string{"message": err.Error()})
	if marshalErr != nil {
		return "{}"
	}
	return string(payload)
}

func isPublicAuthPath(path string) bool {
	switch path {
	case "/auth/bootstrap", "/auth/login", "/auth/register", "/auth/refresh":
		return true
	default:
		return false
	}
}

func (a *DesktopAPI) setTokens(tokens sessionTokens) {
	a.tokensMu.Lock()
	defer a.tokensMu.Unlock()
	a.accessToken = tokens.AccessToken
	a.refreshToken = tokens.RefreshToken
}

func (a *DesktopAPI) clearTokens() {
	a.tokensMu.Lock()
	defer a.tokensMu.Unlock()
	a.accessToken = ""
	a.refreshToken = ""
}

func (a *DesktopAPI) accessTokenValue() string {
	a.tokensMu.RLock()
	defer a.tokensMu.RUnlock()
	return a.accessToken
}

func (a *DesktopAPI) refreshTokenValue() string {
	a.tokensMu.RLock()
	defer a.tokensMu.RUnlock()
	return a.refreshToken
}

func (a *DesktopAPI) hasRefreshToken() bool {
	return a.refreshTokenValue() != ""
}
