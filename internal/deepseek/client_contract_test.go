package deepseek

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientChatRequestContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var request struct {
			Model    string `json:"model"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
			Temperature float64 `json:"temperature"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		assert.Equal(t, "deepseek-test", request.Model)
		require.Len(t, request.Messages, 2)
		assert.Equal(t, "system", request.Messages[0].Role)
		assert.Equal(t, "safe system instruction", request.Messages[0].Content)
		assert.Equal(t, "safe self-reported context", request.Messages[1].Content)
		assert.Equal(t, 0.1, request.Temperature)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"safe response"}}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", "deepseek-test")
	client.HTTPClient = server.Client()
	response, err := client.Chat(context.Background(), "safe system instruction", "safe self-reported context")
	require.NoError(t, err)
	assert.Equal(t, "safe response", response)
}

func TestClientChatProviderErrorMapping(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		expected error
	}{
		{name: "unauthorized", status: http.StatusUnauthorized, expected: ErrInvalidAPIKey},
		{name: "rate limited", status: http.StatusTooManyRequests, expected: ErrRateLimited},
		{name: "unavailable", status: http.StatusServiceUnavailable, expected: ErrServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				if tt.body != "" {
					_, _ = w.Write([]byte(tt.body))
				}
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-key", "deepseek-test")
			client.HTTPClient = server.Client()
			_, err := client.Chat(context.Background(), "system", "message")
			require.Error(t, err)
			assert.True(t, errors.Is(err, tt.expected), "error should preserve the stable provider category")
		})
	}
}

func TestClientChatJSONAcceptsCodeFence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{\"choices\":[{\"message\":{\"content\":\"```json\\n{\\\"message\\\":\\\"safe\\\"}\\n```\"}}]}"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", "deepseek-test")
	client.HTTPClient = server.Client()
	var target struct {
		Message string `json:"message"`
	}
	require.NoError(t, client.ChatJSON(context.Background(), "system", "message", &target))
	assert.Equal(t, "safe", target.Message)
}

func TestClientChatRejectsEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"   "}}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", "deepseek-test")
	client.HTTPClient = server.Client()
	_, err := client.Chat(context.Background(), "system", "message")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmptyResponse))
}

func TestClientChatTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", "deepseek-test")
	client.HTTPClient = &http.Client{Timeout: 20 * time.Millisecond}
	_, err := client.Chat(context.Background(), "system", strings.Repeat("safe", 3))
	require.Error(t, err)
}
