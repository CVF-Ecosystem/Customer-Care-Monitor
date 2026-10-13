package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const telegramAPIBase = "https://api.telegram.org/bot%s/%s"

type TelegramNotifier struct {
	botToken string
	chatID   string
	client   *http.Client
}

func NewTelegramNotifier(botToken, chatID string) *TelegramNotifier {
	return &TelegramNotifier{
		botToken: botToken,
		chatID:   chatID,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (t *TelegramNotifier) Send(ctx context.Context, subject string, body string) error {
	// Combine subject + body with HTML formatting
	text := body
	if subject != "" {
		text = fmt.Sprintf("<b>%s</b>\n\n%s", subject, body)
	}

	// Telegram max message length is 4096
	if len(text) > 4000 {
		text = text[:4000] + "\n\n<i>... (truncated)</i>"
	}

	payload := map[string]interface{}{
		"chat_id":    t.chatID,
		"text":       text,
		"parse_mode": "HTML",
	}

	payloadBytes, _ := json.Marshal(payload)
	url := fmt.Sprintf(telegramAPIBase, t.botToken, "sendMessage")

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payloadBytes))
	if err != nil {
		return failedDelivery("request")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return unknownDelivery("telegram_transport")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return unknownDelivery("telegram_response_body")
	}

	var result map[string]json.RawMessage
	if err := json.Unmarshal(respBody, &result); err != nil || result == nil {
		return unknownDelivery("telegram_response_decode")
	}
	accepted, present := result["ok"]
	if !present {
		return unknownDelivery("telegram_acceptance_missing")
	}
	switch string(bytes.TrimSpace(accepted)) {
	case "true":
		return nil
	case "false":
		return failedDelivery("telegram_rejected")
	default:
		return unknownDelivery("telegram_acceptance_unverifiable")
	}
}

func (t *TelegramNotifier) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf(telegramAPIBase, t.botToken, "getMe")
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("telegram health check request failed")
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram health check transport failed")
	}
	defer resp.Body.Close()

	var result struct {
		OK bool `json:"ok"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("telegram health check response invalid")
	}
	if !result.OK {
		return fmt.Errorf("telegram bot not accessible")
	}
	return nil
}
