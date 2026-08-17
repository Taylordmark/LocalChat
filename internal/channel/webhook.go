package channel

import (
    "bytes"
    "context"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "localchat/internal/store"
)

// Webhook delivers a message by POSTing it, as JSON, to a URL — the
// reference Channel implementation. Config shape:
//
//	{"url": "https://example.com/hook", "secret": "optional-hmac-secret"}
//
// When secret is set, the raw request body is signed HMAC-SHA256 and sent
// as X-LocalChat-Signature (hex-encoded) — the same request-signing
// convention used by Stripe, GitHub, and Twilio, so any receiver already
// built to verify one of those can verify this with a one-line change of
// header name.
type Webhook struct {
    client *http.Client
}

func NewWebhook() *Webhook {
    return &Webhook{client: &http.Client{Timeout: 10 * time.Second}}
}

func (w *Webhook) Type() string { return "webhook" }

// webhookPayload is the generic JSON body a subscriber receives. It carries
// enough of the conversation for the receiver to route without a follow-up
// API call, plus the conversation's own Metadata bag so a caller's
// correlation data (set at conversation-creation time) round-trips back to
// it automatically.
type webhookPayload struct {
    Event        string                 `json:"event"`
    Conversation conversationSummary    `json:"conversation"`
    Message      messageSummary         `json:"message"`
    Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type conversationSummary struct {
    ID    string `json:"id"`
    Title string `json:"title"`
}

type messageSummary struct {
    ID        string    `json:"id"`
    Role      string    `json:"role"`
    Author    string    `json:"author,omitempty"`
    Content   string    `json:"content"`
    CreatedAt time.Time `json:"createdAt"`
}

func (w *Webhook) Deliver(ctx context.Context, conv *store.Conversation, msg *store.Message, config map[string]interface{}) error {
    url, _ := config["url"].(string)
    if url == "" {
        return fmt.Errorf("webhook channel: config.url is required")
    }
    secret, _ := config["secret"].(string)

    body, err := json.Marshal(webhookPayload{
        Event:        "message.created",
        Conversation: conversationSummary{ID: conv.ID, Title: conv.Title},
        Message: messageSummary{
            ID: msg.ID, Role: msg.Role, Author: msg.Author,
            Content: msg.Content, CreatedAt: msg.CreatedAt,
        },
        Metadata: conv.Metadata,
    })
    if err != nil {
        return fmt.Errorf("webhook channel: marshal payload: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
    if err != nil {
        return fmt.Errorf("webhook channel: build request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")
    if secret != "" {
        mac := hmac.New(sha256.New, []byte(secret))
        mac.Write(body)
        req.Header.Set("X-LocalChat-Signature", hex.EncodeToString(mac.Sum(nil)))
    }

    resp, err := w.client.Do(req)
    if err != nil {
        return fmt.Errorf("webhook channel: request failed: %w", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return fmt.Errorf("webhook channel: receiver returned HTTP %d", resp.StatusCode)
    }
    return nil
}
