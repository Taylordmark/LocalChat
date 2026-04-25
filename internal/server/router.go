package server

import (
    "net/http"

    "mychatapp/internal/config"
    "mychatapp/internal/handlers"
    "mychatapp/internal/middleware"

    "github.com/go-chi/chi/v5"
)

func NewRouter(cfg config.Config) http.Handler {
    r := chi.NewRouter()

    r.Use(middleware.Recovery)
    r.Use(middleware.Logging)
    r.Use(middleware.CORS)

    r.Get("/health", handlers.Health)

    r.Post("/api/chat", handlers.Chat(cfg))

    // Serve frontend
    r.Handle("/*", http.FileServer(http.Dir("web")))

    return r
}
