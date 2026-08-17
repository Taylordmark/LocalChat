package store

import (
    "context"
    "crypto/rand"
    "crypto/sha256"
    "database/sql"
    "encoding/hex"
    "errors"
    "time"
)

// APIKey is a credential for the external Conversations API. The raw
// secret is never stored — only its SHA-256 hash — the same convention
// used across most REST APIs with bearer credentials.
type APIKey struct {
    ID         string     `json:"id"`
    Name       string     `json:"name"`
    CreatedAt  time.Time  `json:"createdAt"`
    LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
}

// GenerateAPIKeySecret returns a new random secret in the form
// "lc_<64 hex chars>" — a prefixed opaque token, the same shape used by
// Stripe/OpenAI/GitHub tokens, so a leaked key is instantly recognizable
// in logs or source-control scanners.
func GenerateAPIKeySecret() string {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        panic("store: crypto/rand unavailable: " + err.Error())
    }
    return "lc_" + hex.EncodeToString(b)
}

func hashKey(secret string) string {
    sum := sha256.Sum256([]byte(secret))
    return hex.EncodeToString(sum[:])
}

// CreateAPIKey mints and stores a new key, returning both the DB record
// and the one-time raw secret — the only moment the caller can see it.
func (s *Store) CreateAPIKey(ctx context.Context, name string) (*APIKey, string, error) {
    secret := GenerateAPIKeySecret()
    k := &APIKey{ID: newID("key"), Name: name, CreatedAt: time.Now().UTC()}
    _, err := s.db.ExecContext(ctx,
        `INSERT INTO api_keys (id, name, key_hash, created_at) VALUES (?, ?, ?, ?)`,
        k.ID, k.Name, hashKey(secret), k.CreatedAt.Format(time.RFC3339Nano))
    if err != nil {
        return nil, "", err
    }
    return k, secret, nil
}

// ValidateAPIKey looks up a raw secret by its hash and, on success, stamps
// last_used_at. Returns ErrNotFound for an unknown or revoked key.
func (s *Store) ValidateAPIKey(ctx context.Context, secret string) (*APIKey, error) {
    h := hashKey(secret)
    row := s.db.QueryRowContext(ctx,
        `SELECT id, name, created_at, last_used_at FROM api_keys WHERE key_hash = ?`, h)
    var k APIKey
    var createdAt string
    var lastUsed sql.NullString
    if err := row.Scan(&k.ID, &k.Name, &createdAt, &lastUsed); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrNotFound
        }
        return nil, err
    }
    k.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
    if lastUsed.Valid {
        if t, err := time.Parse(time.RFC3339Nano, lastUsed.String); err == nil {
            k.LastUsedAt = &t
        }
    }

    now := time.Now().UTC()
    _, _ = s.db.ExecContext(ctx, `UPDATE api_keys SET last_used_at = ? WHERE id = ?`, now.Format(time.RFC3339Nano), k.ID)
    return &k, nil
}

func (s *Store) ListAPIKeys(ctx context.Context) ([]*APIKey, error) {
    rows, err := s.db.QueryContext(ctx, `SELECT id, name, created_at, last_used_at FROM api_keys ORDER BY created_at DESC`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var out []*APIKey
    for rows.Next() {
        var k APIKey
        var createdAt string
        var lastUsed sql.NullString
        if err := rows.Scan(&k.ID, &k.Name, &createdAt, &lastUsed); err != nil {
            return nil, err
        }
        k.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
        if lastUsed.Valid {
            if t, err := time.Parse(time.RFC3339Nano, lastUsed.String); err == nil {
                k.LastUsedAt = &t
            }
        }
        out = append(out, &k)
    }
    return out, rows.Err()
}

func (s *Store) RevokeAPIKey(ctx context.Context, id string) error {
    res, err := s.db.ExecContext(ctx, `DELETE FROM api_keys WHERE id = ?`, id)
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

// CountAPIKeys is used at startup to decide whether a bootstrap admin key
// needs to be minted.
func (s *Store) CountAPIKeys(ctx context.Context) (int, error) {
    var n int
    err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM api_keys`).Scan(&n)
    return n, err
}
