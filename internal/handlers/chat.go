package handlers

import (
    "encoding/json"
    "localchat/internal/config"
    "localchat/internal/ollama"
    "net/http"
)

type ChatRequest struct {
    Model    string           `json:"model"`
    Messages []ollama.Message `json:"messages"`
}

func Chat(cfg config.Config) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req ChatRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "invalid request", http.StatusBadRequest)
            return
        }

        if req.Model == "" {
            req.Model = "llama3:8b"
        }

        w.Header().Set("Content-Type", "text/event-stream")

        stream, err := ollama.ChatStream(cfg.OllamaHost, req.Model, req.Messages)
        if err != nil {
            http.Error(w, "failed to connect to ollama", http.StatusInternalServerError)
            return
        }
        defer stream.Close()

        buf := make([]byte, 4096)
        for {
            n, err := stream.Read(buf)
            if n > 0 {
                w.Write(buf[:n])
                w.(http.Flusher).Flush()
            }
            if err != nil {
                break
            }
        }
    }
}
