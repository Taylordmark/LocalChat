package main

import (
    "context"
    "log"
    "net/http"

    "localchat/internal/apiauth"
    "localchat/internal/channel"
    "localchat/internal/config"
    "localchat/internal/handlers"
    "localchat/internal/server"
    "localchat/internal/store"
)

func main() {
    cfg := config.Load()

    db, err := store.Open(cfg.DBPath)
    if err != nil {
        log.Fatalf("failed to open store: %v", err)
    }
    defer db.Close()

    if err := apiauth.Bootstrap(context.Background(), db); err != nil {
        log.Fatalf("failed to bootstrap API key: %v", err)
    }

    dispatcher := channel.NewDispatcher(db, channel.NewWebhook())
    api := &handlers.ConversationsAPI{Store: db, Dispatcher: dispatcher}

    r := server.NewRouter(cfg, api)

    log.Printf("Server running on %s", cfg.ServerAddr)
    if err := http.ListenAndServe(cfg.ServerAddr, r); err != nil {
        log.Fatalf("server error: %v", err)
    }
}
