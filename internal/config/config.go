package config

import (
    "os"
)

type Config struct {
    ServerAddr string
    OllamaHost string
    DBPath     string
}

func Load() Config {
    addr := os.Getenv("SERVER_ADDR")
    if addr == "" {
        addr = ":8080"
    }

    ollama := os.Getenv("OLLAMA_HOST")
    if ollama == "" {
        ollama = "http://ollama:11434"
    }

    dbPath := os.Getenv("LOCALCHAT_DB_PATH")
    if dbPath == "" {
        dbPath = "localchat.db"
    }

    return Config{
        ServerAddr: addr,
        OllamaHost: ollama,
        DBPath:     dbPath,
    }
}
