package handlers

import (
    "io"
    "net/http"

    "localchat/internal/config"
)

func ListModels(cfg config.Config) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        resp, err := http.Get(cfg.OllamaHost + "/api/tags")
        if err != nil {
            http.Error(w, err.Error(), 500)
            return
        }
        defer resp.Body.Close()

        w.Header().Set("Content-Type", "application/json")
        io.Copy(w, resp.Body)
    }
}
