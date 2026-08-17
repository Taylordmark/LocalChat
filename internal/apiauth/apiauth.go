// Package apiauth gates the external Conversations API behind API keys —
// the standard bearer-credential pattern used by every B2B REST API
// (Stripe, Twilio, OpenAI). It is deliberately separate from, and does not
// touch, the existing local browser UI's endpoints (/api/chat, /api/models,
// /health), which stay open — this only protects the new /api/v1 surface
// meant for external callers.
package apiauth

import (
    "context"
    "log"
    "net/http"
    "strings"

    "localchat/internal/store"
)

type contextKey string

const keyContextKey contextKey = "apiauth.key"

// Middleware requires a valid API key on every request, supplied either as
// "Authorization: Bearer <key>" or "X-API-Key: <key>". On success the
// validated key is attached to the request context (see KeyFromContext).
func Middleware(s *store.Store) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            secret := extractKey(r)
            if secret == "" {
                http.Error(w, `{"error":"missing API key"}`, http.StatusUnauthorized)
                return
            }
            key, err := s.ValidateAPIKey(r.Context(), secret)
            if err != nil {
                if err != store.ErrNotFound {
                    log.Printf("apiauth: validation error: %v", err)
                }
                http.Error(w, `{"error":"invalid API key"}`, http.StatusUnauthorized)
                return
            }
            ctx := context.WithValue(r.Context(), keyContextKey, key)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func extractKey(r *http.Request) string {
    if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
        return strings.TrimPrefix(h, "Bearer ")
    }
    return r.Header.Get("X-API-Key")
}

// KeyFromContext returns the API key that authenticated this request, if
// any (always present on requests that passed Middleware).
func KeyFromContext(ctx context.Context) *store.APIKey {
    k, _ := ctx.Value(keyContextKey).(*store.APIKey)
    return k
}

// Bootstrap mints an initial admin API key the first time the server ever
// starts (no keys exist yet) and prints it once to the log — the same
// pattern Jenkins, Grafana, and similar self-hosted tools use to hand an
// operator their first credential without a signup flow. That key can then
// mint additional named keys for each integration via the API itself.
func Bootstrap(ctx context.Context, s *store.Store) error {
    n, err := s.CountAPIKeys(ctx)
    if err != nil {
        return err
    }
    if n > 0 {
        return nil
    }
    _, secret, err := s.CreateAPIKey(ctx, "bootstrap")
    if err != nil {
        return err
    }
    log.Printf("=======================================================================")
    log.Printf("No API keys found — minted a bootstrap admin key (shown once):")
    log.Printf("  %s", secret)
    log.Printf("Use it as 'Authorization: Bearer %s' to call the Conversations API,", secret)
    log.Printf("or paste it into the web UI's Conversations tab to mint named keys for")
    log.Printf("other integrations from there.")
    log.Printf("=======================================================================")
    return nil
}
