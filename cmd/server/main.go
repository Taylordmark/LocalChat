package main

import (
    "log"
    "net/http"

    "mychatapp/internal/config"
    "mychatapp/internal/server"
)

func main() {
    cfg := config.Load()

    r := server.NewRouter(cfg)

    log.Printf("Server running on %s", cfg.ServerAddr)
    if err := http.ListenAndServe(cfg.ServerAddr, r); err != nil {
        log.Fatalf("server error: %v", err)
    }
}
