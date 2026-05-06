package ollama

import (
    "bytes"
    "encoding/json"
    "io"
    "net/http"
)

type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type ChatRequest struct {
    Model    string    `json:"model"`
    Messages []Message `json:"messages"`
}

func ChatStream(host, model string, messages []Message) (io.ReadCloser, error) {
    req := ChatRequest{
        Model:    model,
        Messages: messages,
    }

    body, _ := json.Marshal(req)

    httpReq, err := http.NewRequest("POST", host+"/api/chat", bytes.NewReader(body))
    if err != nil {
        return nil, err
    }

    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := http.DefaultClient.Do(httpReq)
    if err != nil {
        return nil, err
    }

    return resp.Body, nil
}
