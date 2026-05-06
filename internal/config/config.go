package config

import (
    "os"
)

type Config struct {
    ServerAddr string
    OllamaHost string
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

    return Config{
        ServerAddr: addr,
        OllamaHost: ollama,
    }
}
