package handlers

import (
    "encoding/json"
    "net/http"

    "github.com/go-chi/chi/v5"

    "localchat/internal/channel"
    "localchat/internal/store"
)

// ConversationsAPI holds the dependencies for the external Conversations
// API — the generic, product-level surface any external system integrates
// against, independent of who that system is.
type ConversationsAPI struct {
    Store      *store.Store
    Dispatcher *channel.Dispatcher
}

type createConversationRequest struct {
    Title    string                 `json:"title"`
    Metadata map[string]interface{} `json:"metadata"`
    // Message and Channel are one-shot convenience fields — most callers
    // creating a conversation immediately want to post an opening message
    // and bind a reply destination, so this saves the two follow-up calls
    // (a pattern also offered by Intercom's and Front's conversation-create
    // endpoints). Both are equally reachable via their own dedicated
    // endpoints below for callers that want to compose the steps.
    Message *createMessageRequest `json:"message,omitempty"`
    Channel *createChannelRequest `json:"channel,omitempty"`
}

type createMessageRequest struct {
    Role    string `json:"role"`
    Author  string `json:"author"`
    Content string `json:"content"`
}

func (a *ConversationsAPI) CreateConversation(w http.ResponseWriter, r *http.Request) {
    var req createConversationRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid JSON body")
        return
    }

    conv, err := a.Store.CreateConversation(r.Context(), req.Title, req.Metadata)
    if err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }

    if req.Channel != nil {
        if req.Channel.Type == "" {
            writeError(w, http.StatusBadRequest, "conversation created, but channel.type is required")
            return
        }
        if _, err := a.Store.CreateChannel(r.Context(), conv.ID, req.Channel.Type, req.Channel.Config); err != nil {
            writeError(w, http.StatusBadRequest, "conversation created, but channel binding failed: "+err.Error())
            return
        }
    }

    if req.Message != nil && req.Message.Content != "" {
        role := req.Message.Role
        if role == "" {
            role = "user"
        }
        msg, err := a.Store.AddMessage(r.Context(), conv.ID, role, req.Message.Author, req.Message.Content)
        if err != nil {
            writeError(w, http.StatusBadRequest, "conversation created, but initial message failed: "+err.Error())
            return
        }
        a.Dispatcher.Dispatch(r.Context(), conv, msg)
    }

    writeJSON(w, http.StatusCreated, conv)
}

func (a *ConversationsAPI) ListConversations(w http.ResponseWriter, r *http.Request) {
    convs, err := a.Store.ListConversations(r.Context(), 0)
    if err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }
    if convs == nil {
        convs = []*store.Conversation{}
    }
    writeJSON(w, http.StatusOK, convs)
}

type conversationWithMessages struct {
    *store.Conversation
    Messages []*store.Message `json:"messages"`
}

func (a *ConversationsAPI) GetConversation(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    conv, err := a.Store.GetConversation(r.Context(), id)
    if err != nil {
        writeNotFoundOr500(w, err)
        return
    }
    msgs, err := a.Store.ListMessages(r.Context(), id)
    if err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }
    if msgs == nil {
        msgs = []*store.Message{}
    }
    writeJSON(w, http.StatusOK, conversationWithMessages{Conversation: conv, Messages: msgs})
}

func (a *ConversationsAPI) PostMessage(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    var req createMessageRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid JSON body")
        return
    }
    if req.Content == "" {
        writeError(w, http.StatusBadRequest, "content is required")
        return
    }
    if req.Role == "" {
        req.Role = "user"
    }

    conv, err := a.Store.GetConversation(r.Context(), id)
    if err != nil {
        writeNotFoundOr500(w, err)
        return
    }
    msg, err := a.Store.AddMessage(r.Context(), id, req.Role, req.Author, req.Content)
    if err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }

    // Delivery is fire-and-forget from the API's perspective (see
    // channel.Dispatcher) — this returns as soon as the message is
    // durably stored, not after every bound channel has been reached.
    a.Dispatcher.Dispatch(r.Context(), conv, msg)
    writeJSON(w, http.StatusCreated, msg)
}

func (a *ConversationsAPI) ListMessages(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    if _, err := a.Store.GetConversation(r.Context(), id); err != nil {
        writeNotFoundOr500(w, err)
        return
    }
    msgs, err := a.Store.ListMessages(r.Context(), id)
    if err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }
    if msgs == nil {
        msgs = []*store.Message{}
    }
    writeJSON(w, http.StatusOK, msgs)
}
