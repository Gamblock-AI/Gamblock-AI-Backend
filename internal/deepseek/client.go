package deepseek

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	model      string
	HTTPClient *http.Client
}

func NewClient(baseURL, apiKey, model string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		model:      model,
		HTTPClient: &http.Client{Timeout: 60 * time.Second},
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type chatError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

var (
	ErrEmptyResponse    = errors.New("deepseek: empty response content")
	ErrInvalidAPIKey    = errors.New("deepseek: invalid or missing api key")
	ErrRateLimited      = errors.New("deepseek: rate limited, please wait")
	ErrServiceUnavailable = errors.New("deepseek: service temporarily unavailable")
)

func (c *Client) Translate(ctx context.Context, text string, sourceLang, targetLang string) (string, error) {
	systemPrompt := fmt.Sprintf(
		"You are a professional translator. Translate the user's text from %s to %s. "+
			"Return ONLY the translation without any commentary, notes, quotation marks, or formatting. "+
			"Preserve the original tone, register, and meaning as accurately as possible. "+
			"If the input is already in the target language, return it unchanged.",
		languageName(sourceLang), languageName(targetLang),
	)
	return c.Chat(ctx, systemPrompt, text)
}

func (c *Client) Chat(ctx context.Context, systemPrompt string, userMessage string) (string, error) {	if strings.TrimSpace(c.apiKey) == "" {
		return "", ErrInvalidAPIKey
	}

	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMessage},
		},
		Temperature: 0.1,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("deepseek: marshal request: %w", err)
	}

	url := c.baseURL + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("deepseek: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("deepseek: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("deepseek: read response: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		// OK — continue parsing
	case http.StatusTooManyRequests:
		return "", ErrRateLimited
	case http.StatusUnauthorized:
		return "", ErrInvalidAPIKey
	case http.StatusServiceUnavailable, http.StatusBadGateway, http.StatusGatewayTimeout:
		return "", ErrServiceUnavailable
	default:
		var errResp chatError
		if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
			return "", fmt.Errorf("deepseek: api error (%d): %s", resp.StatusCode, errResp.Error.Message)
		}
		return "", fmt.Errorf("deepseek: unexpected status %d", resp.StatusCode)
	}

	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("deepseek: unmarshal response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", ErrEmptyResponse
	}

	content := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	if content == "" {
		return "", ErrEmptyResponse
	}

	return content, nil
}

// ChatJSON runs a chat completion and unmarshals the response into target,
// tolerating a ```json fence wrapper. Temperature stays at the deterministic
// 0.1 value used across the codebase.
func (c *Client) ChatJSON(ctx context.Context, systemPrompt, userMessage string, target any) error {
	raw, err := c.Chat(ctx, systemPrompt, userMessage)
	if err != nil {
		return err
	}
	clean := strings.TrimSpace(raw)
	clean = strings.TrimPrefix(clean, "```json")
	clean = strings.TrimPrefix(clean, "```")
	clean = strings.TrimSuffix(clean, "```")
	clean = strings.TrimSpace(clean)
	if err := json.Unmarshal([]byte(clean), target); err != nil {
		return fmt.Errorf("deepseek: parse json response: %w", err)
	}
	return nil
}

func languageName(code string) string {
	switch code {
	case "id":
		return "Bahasa Indonesia"
	case "en":
		return "English"
	default:
		return code
	}
}
