package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	DefaultOllamaURL = "http://127.0.0.1:11434"
	DefaultModel     = "llama3.2"
)

// Ollama is a Provider backed by a local Ollama server (/api/generate).
type Ollama struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewOllama returns an Ollama provider; empty arguments use the defaults.
func NewOllama(baseURL, model string) *Ollama {
	if baseURL == "" {
		baseURL = DefaultOllamaURL
	}
	if model == "" {
		model = DefaultModel
	}
	return &Ollama{
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Ollama rejects a schema sent as a JSON string; it must be a JSON object,
// hence json.RawMessage rather than string.
type generateRequest struct {
	Model   string          `json:"model"`
	Prompt  string          `json:"prompt"`
	Stream  bool            `json:"stream"`
	Format  json.RawMessage `json:"format,omitempty"`
	Options generateOptions `json:"options"`
}

type generateOptions struct {
	Temperature float64 `json:"temperature"`
}

type generateResponse struct {
	Response string `json:"response"`
}

// Generate implements Provider. Temperature is fixed at 0 so the same prompt
// gives the same answer, which keeps caching and evaluation meaningful.
func (o *Ollama) Generate(ctx context.Context, prompt, schema string) ([]byte, error) {
	req := generateRequest{
		Model:  o.model,
		Prompt: prompt,
		Stream: false,
	}
	if schema != "" {
		if !json.Valid([]byte(schema)) {
			return nil, fmt.Errorf("ollama: schema is not valid JSON")
		}
		req.Format = json.RawMessage(schema)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama status %d: %s", resp.StatusCode, trimBody(raw))
	}

	var out generateResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("ollama decode: %w", err)
	}
	return []byte(out.Response), nil
}

func trimBody(b []byte) string {
	const max = 200
	s := string(b)
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
