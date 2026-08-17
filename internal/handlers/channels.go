package handlers

import (
    "encoding/json"
    "net/http"

    "github.com/go-chi/chi/v5"

    "localchat/internal/store"
)

type createChannelRequest struct {
    // ConversationID is optional — omit it to create a global channel that
    // fires for every conversation (see store.Channel).
    ConversationID string                 `json:"conversationId,omitempty"`
    Type           string                 `json:"type"`
    Config         map[string]interface{} `json:"config"`
}

func (a *ConversationsAPI) CreateChannel(w http.ResponseWriter, r *http.Request) {
    var req createChannelRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid JSON body")
        return
    }
    if req.Type == "" {
        writeError(w, http.StatusBadRequest, "type is required")
        return
    }
    ch, err := a.Store.CreateChannel(r.Context(), req.ConversationID, req.Type, req.Config)
    if err != nil {
        writeNotFoundOr500(w, err)
        return
    }
    writeJSON(w, http.StatusCreated, ch)
}

func (a *ConversationsAPI) ListChannels(w http.ResponseWriter, r *http.Request) {
    chs, err := a.Store.ListChannels(r.Context())
    if err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }
    if chs == nil {
        chs = []*store.Channel{}
    }
    writeJSON(w, http.StatusOK, chs)
}

func (a *ConversationsAPI) DeleteChannel(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    if err := a.Store.DeleteChannel(r.Context(), id); err != nil {
        writeNotFoundOr500(w, err)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}
