package ollama

type GenerateRequest struct {
    Model  string `json:"model"`
    Prompt string `json:"prompt"`
    Stream bool   `json:"stream"`
}

type GenerateResponse struct {
    Model     string `json:"model,omitempty"`
    CreatedAt string `json:"created_at,omitempty"`
    Response  string `json:"response,omitempty"`
    Done      bool   `json:"done,omitempty"`
}
