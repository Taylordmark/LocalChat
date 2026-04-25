package ollama

import (
    "bufio"
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

// StreamGenerate sends a streaming request to Ollama and writes streamed chunks to w.
func StreamGenerate(host, model, prompt string, w io.Writer) {
    body, _ := json.Marshal(GenerateRequest{
        Model:  model,
        Prompt: prompt,
        Stream: true,
    })

    req, _ := http.NewRequest("POST", host+"/api/generate", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        fmt.Fprintf(w, "error: %v\n", err)
        return
    }
    defer resp.Body.Close()

    scanner := bufio.NewScanner(resp.Body)
    for scanner.Scan() {
        w.Write(scanner.Bytes())
        w.Write([]byte("\n"))
    }
}
