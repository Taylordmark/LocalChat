package store

import (
    "context"
    "database/sql"
    "encoding/json"
    "errors"
    "time"
)

var ErrNotFound = errors.New("not found")

// Conversation is a thread of messages between any number of parties — a
// human in localchat's own UI, an external system posting on behalf of a
// customer, a delivery channel relaying an SMS/email reply, and so on.
// localchat has no opinion on who the parties are; Metadata lets a caller
// attach its own correlation data (e.g. an external record ID) without
// localchat needing to understand it.
type Conversation struct {
    ID        string                 `json:"id"`
    Title     string                 `json:"title"`
    Metadata  map[string]interface{} `json:"metadata"`
    CreatedAt time.Time              `json:"createdAt"`
    UpdatedAt time.Time              `json:"updatedAt"`
}

// Message is one entry in a Conversation. Role and Author are free text —
// localchat imposes no fixed participant model ("user"/"assistant"/"agent"/
// a phone number/an external system name are all valid) so any integration
// can express its own participants naturally.
type Message struct {
    ID             string    `json:"id"`
    ConversationID string    `json:"conversationId"`
    Role           string    `json:"role"`
    Author         string    `json:"author,omitempty"`
    Content        string    `json:"content"`
    CreatedAt      time.Time `json:"createdAt"`
}

func (s *Store) CreateConversation(ctx context.Context, title string, metadata map[string]interface{}) (*Conversation, error) {
    if metadata == nil {
        metadata = map[string]interface{}{}
    }
    metaJSON, err := json.Marshal(metadata)
    if err != nil {
        return nil, err
    }
    now := time.Now().UTC()
    c := &Conversation{ID: newID("conv"), Title: title, Metadata: metadata, CreatedAt: now, UpdatedAt: now}
    _, err = s.db.ExecContext(ctx,
        `INSERT INTO conversations (id, title, metadata_json, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
        c.ID, c.Title, string(metaJSON), c.CreatedAt.Format(time.RFC3339Nano), c.UpdatedAt.Format(time.RFC3339Nano))
    if err != nil {
        return nil, err
    }
    return c, nil
}

func (s *Store) GetConversation(ctx context.Context, id string) (*Conversation, error) {
    row := s.db.QueryRowContext(ctx,
        `SELECT id, title, metadata_json, created_at, updated_at FROM conversations WHERE id = ?`, id)
    return scanConversation(row)
}

// ListConversations returns conversations newest-first. limit<=0 means "all".
func (s *Store) ListConversations(ctx context.Context, limit int) ([]*Conversation, error) {
    q := `SELECT id, title, metadata_json, created_at, updated_at FROM conversations ORDER BY updated_at DESC`
    args := []interface{}{}
    if limit > 0 {
        q += ` LIMIT ?`
        args = append(args, limit)
    }
    rows, err := s.db.QueryContext(ctx, q, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var out []*Conversation
    for rows.Next() {
        c, err := scanConversation(rows)
        if err != nil {
            return nil, err
        }
        out = append(out, c)
    }
    return out, rows.Err()
}

func scanConversation(row interface {
    Scan(dest ...interface{}) error
}) (*Conversation, error) {
    var c Conversation
    var metaJSON, createdAt, updatedAt string
    if err := row.Scan(&c.ID, &c.Title, &metaJSON, &createdAt, &updatedAt); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrNotFound
        }
        return nil, err
    }
    if err := json.Unmarshal([]byte(metaJSON), &c.Metadata); err != nil {
        c.Metadata = map[string]interface{}{}
    }
    c.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
    c.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
    return &c, nil
}

// AddMessage appends a message to a conversation and bumps the
// conversation's updated_at so ListConversations reflects recent activity.
func (s *Store) AddMessage(ctx context.Context, conversationID, role, author, content string) (*Message, error) {
    if _, err := s.GetConversation(ctx, conversationID); err != nil {
        return nil, err
    }
    now := time.Now().UTC()
    m := &Message{ID: newID("msg"), ConversationID: conversationID, Role: role, Author: author, Content: content, CreatedAt: now}

    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback() //nolint:errcheck

    if _, err := tx.ExecContext(ctx,
        `INSERT INTO messages (id, conversation_id, role, author, content, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
        m.ID, m.ConversationID, m.Role, m.Author, m.Content, m.CreatedAt.Format(time.RFC3339Nano)); err != nil {
        return nil, err
    }
    if _, err := tx.ExecContext(ctx, `UPDATE conversations SET updated_at = ? WHERE id = ?`,
        now.Format(time.RFC3339Nano), conversationID); err != nil {
        return nil, err
    }
    if err := tx.Commit(); err != nil {
        return nil, err
    }
    return m, nil
}

func (s *Store) ListMessages(ctx context.Context, conversationID string) ([]*Message, error) {
    rows, err := s.db.QueryContext(ctx,
        `SELECT id, conversation_id, role, author, content, created_at FROM messages WHERE conversation_id = ? ORDER BY created_at ASC`,
        conversationID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var out []*Message
    for rows.Next() {
        var m Message
        var createdAt string
        if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Author, &m.Content, &createdAt); err != nil {
            return nil, err
        }
        m.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
        out = append(out, &m)
    }
    return out, rows.Err()
}
