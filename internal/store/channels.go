package store

import (
    "context"
    "database/sql"
    "encoding/json"
    "time"
)

// Channel is a delivery destination bound to a conversation (or, when
// ConversationID is empty, to every conversation). Each message posted
// fires every applicable channel — see package channel for what "fires"
// means for a given Type. Config is opaque to the store; it's whatever
// shape the channel type needs (e.g. a webhook channel's {"url","secret"}).
type Channel struct {
    ID             string                 `json:"id"`
    ConversationID string                 `json:"conversationId,omitempty"`
    Type           string                 `json:"type"`
    Config         map[string]interface{} `json:"config"`
    CreatedAt      time.Time              `json:"createdAt"`
}

func (s *Store) CreateChannel(ctx context.Context, conversationID, channelType string, config map[string]interface{}) (*Channel, error) {
    if conversationID != "" {
        if _, err := s.GetConversation(ctx, conversationID); err != nil {
            return nil, err
        }
    }
    if config == nil {
        config = map[string]interface{}{}
    }
    cfgJSON, err := json.Marshal(config)
    if err != nil {
        return nil, err
    }
    ch := &Channel{ID: newID("chan"), ConversationID: conversationID, Type: channelType, Config: config, CreatedAt: time.Now().UTC()}

    var convArg interface{}
    if conversationID != "" {
        convArg = conversationID
    } // else leaves convArg nil -> SQL NULL, the "global channel" marker
    _, err = s.db.ExecContext(ctx,
        `INSERT INTO channels (id, conversation_id, type, config_json, created_at) VALUES (?, ?, ?, ?, ?)`,
        ch.ID, convArg, ch.Type, string(cfgJSON), ch.CreatedAt.Format(time.RFC3339Nano))
    if err != nil {
        return nil, err
    }
    return ch, nil
}

// ChannelsFor returns every channel that should fire for conversationID:
// channels bound specifically to it, plus every global channel (NULL
// conversation_id).
func (s *Store) ChannelsFor(ctx context.Context, conversationID string) ([]*Channel, error) {
    rows, err := s.db.QueryContext(ctx,
        `SELECT id, conversation_id, type, config_json, created_at FROM channels
         WHERE conversation_id = ? OR conversation_id IS NULL`, conversationID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    return scanChannels(rows)
}

func (s *Store) ListChannels(ctx context.Context) ([]*Channel, error) {
    rows, err := s.db.QueryContext(ctx,
        `SELECT id, conversation_id, type, config_json, created_at FROM channels ORDER BY created_at DESC`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    return scanChannels(rows)
}

func scanChannels(rows *sql.Rows) ([]*Channel, error) {
    var out []*Channel
    for rows.Next() {
        var ch Channel
        var convID sql.NullString
        var cfgJSON, createdAt string
        if err := rows.Scan(&ch.ID, &convID, &ch.Type, &cfgJSON, &createdAt); err != nil {
            return nil, err
        }
        ch.ConversationID = convID.String
        if err := json.Unmarshal([]byte(cfgJSON), &ch.Config); err != nil {
            ch.Config = map[string]interface{}{}
        }
        ch.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
        out = append(out, &ch)
    }
    return out, rows.Err()
}

func (s *Store) DeleteChannel(ctx context.Context, id string) error {
    res, err := s.db.ExecContext(ctx, `DELETE FROM channels WHERE id = ?`, id)
    if err != nil {
        return err
    }
    n, err := res.RowsAffected()
    if err != nil {
        return err
    }
    if n == 0 {
        return ErrNotFound
    }
    return nil
}
