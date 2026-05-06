package handlers

import (
    "localchat/internal/config"
    "localchat/internal/ollama"
    "encoding/json"
    "net/http"
)

type ChatRequest struct {
    Model  string `json:"model"`
    Prompt string `json:"prompt"`
}

func Chat(cfg config.Config) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req ChatRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "invalid request", http.StatusBadRequest)
            return
        }

        w.Header().Set("Content-Type", "text/event-stream")

        ollama.StreamGenerate(cfg.OllamaHost, req.Model, req.Prompt, w)
    }
}
