package handlers

import (
    "encoding/json"
    "net/http"

    "github.com/go-chi/chi/v5"

    "localchat/internal/apiauth"
    "localchat/internal/store"
)

type createKeyRequest struct {
    Name string `json:"name"`
}

type createKeyResponse struct {
    *store.APIKey
    // Secret is only ever present in this one response — it is not
    // recoverable afterward, matching how every major API key system
    // (Stripe, GitHub, AWS) shows a new secret exactly once.
    Secret string `json:"secret"`
}

func (a *ConversationsAPI) CreateKey(w http.ResponseWriter, r *http.Request) {
    var req createKeyRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid JSON body")
        return
    }
    if req.Name == "" {
        writeError(w, http.StatusBadRequest, "name is required")
        return
    }
    key, secret, err := a.Store.CreateAPIKey(r.Context(), req.Name)
    if err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }
    writeJSON(w, http.StatusCreated, createKeyResponse{APIKey: key, Secret: secret})
}

func (a *ConversationsAPI) ListKeys(w http.ResponseWriter, r *http.Request) {
    keys, err := a.Store.ListAPIKeys(r.Context())
    if err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }
    if keys == nil {
        keys = []*store.APIKey{}
    }
    writeJSON(w, http.StatusOK, keys)
}

func (a *ConversationsAPI) RevokeKey(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    if err := a.Store.RevokeAPIKey(r.Context(), id); err != nil {
        writeNotFoundOr500(w, err)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}

// WhoAmI reports which key authenticated the current request — a small,
// generically useful endpoint (every credentialed API benefits from one)
// that lets a caller (or the web UI, after pasting in a key) confirm the
// key is valid and see its own name without a separate lookup.
func (a *ConversationsAPI) WhoAmI(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, http.StatusOK, apiauth.KeyFromContext(r.Context()))
}
