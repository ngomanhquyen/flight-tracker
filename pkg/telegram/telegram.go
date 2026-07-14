// Package telegram is a minimal client for the parts of the Telegram Bot
// API this project needs (receiving webhook updates, sending messages,
// registering the webhook). It intentionally does not pull in a
// third-party Telegram SDK, keeping the dependency surface aligned with
// the project's chosen stack.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const defaultAPIBase = "https://api.telegram.org"

// Update is the subset of Telegram's Update object this project consumes.
// https://core.telegram.org/bots/api#update
type Update struct {
	UpdateID int      `json:"update_id"`
	Message  *Message `json:"message"`
}

type Message struct {
	MessageID int    `json:"message_id"`
	From      *User  `json:"from"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
}

type User struct {
	ID           int64  `json:"id"`
	IsBot        bool   `json:"is_bot"`
	Username     string `json:"username"`
	FirstName    string `json:"first_name"`
	LanguageCode string `json:"language_code"`
}

type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

// Sender is the outbound capability bot-service and notification-service
// both depend on, so either can be tested against a fake without touching
// the network.
type Sender interface {
	SendMessage(ctx context.Context, chatID int64, text string) error
}

// Client is a thin, timeout-bound HTTP wrapper around the Bot API.
type Client struct {
	httpClient *http.Client
	apiBase    string
	botToken   string
}

func NewClient(botToken string, timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		apiBase:    defaultAPIBase,
		botToken:   botToken,
	}
}

func (c *Client) endpoint(method string) string {
	return fmt.Sprintf("%s/bot%s/%s", c.apiBase, c.botToken, method)
}

type apiResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
}

// SendMessage sends a plain-text message to chatID.
func (c *Client) SendMessage(ctx context.Context, chatID int64, text string) error {
	body, err := json.Marshal(map[string]any{
		"chat_id": chatID,
		"text":    text,
	})
	if err != nil {
		return fmt.Errorf("telegram: marshal sendMessage body: %w", err)
	}
	return c.post(ctx, "sendMessage", body)
}

// SetWebhook registers webhookURL with Telegram, protected by secretToken
// (delivered back on every update via the
// X-Telegram-Bot-Api-Secret-Token header).
func (c *Client) SetWebhook(ctx context.Context, webhookURL, secretToken string) error {
	body, err := json.Marshal(map[string]any{
		"url":          webhookURL,
		"secret_token": secretToken,
	})
	if err != nil {
		return fmt.Errorf("telegram: marshal setWebhook body: %w", err)
	}
	return c.post(ctx, "setWebhook", body)
}

func (c *Client) post(ctx context.Context, method string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(method), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: call %s: %w", method, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var parsed apiResponse
	_ = json.Unmarshal(respBody, &parsed)

	if resp.StatusCode != http.StatusOK || !parsed.OK {
		return fmt.Errorf("telegram: %s failed (status %d): %s", method, resp.StatusCode, parsed.Description)
	}
	return nil
}

// WebhookURL joins a base host with the per-deployment path secret, e.g.
// https://bot.example.com + "abc123" -> https://bot.example.com/webhook/telegram/abc123
func WebhookURL(baseURL, pathSecret string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("telegram: invalid base URL: %w", err)
	}
	u.Path = fmt.Sprintf("/webhook/telegram/%s", pathSecret)
	return u.String(), nil
}
