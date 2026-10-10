package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var _ Provider = (*Ollama)(nil)

const testSchema = `{"type":"object","properties":{"summary":{"type":"string"}}}`

// fakeOllama records the decoded request body and replies with response.
func fakeOllama(t *testing.T, status int, response string) (*httptest.Server, *map[string]any) {
	t.Helper()
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(srv.Close)
	return srv, &got
}

func TestOllamaGenerateSendsSchemaAsObject(t *testing.T) {
	t.Parallel()
	srv, got := fakeOllama(t, http.StatusOK, `{"response":"{\"summary\":\"hi\"}"}`)

	out, err := NewOllama(srv.URL, "test-model").Generate(context.Background(), "say hi", testSchema)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != `{"summary":"hi"}` {
		t.Fatalf("response = %s", out)
	}

	req := *got
	if req["model"] != "test-model" || req["prompt"] != "say hi" || req["stream"] != false {
		t.Fatalf("request = %+v", req)
	}
	if _, ok := req["format"].(map[string]any); !ok {
		t.Fatalf("format must be a JSON object, got %T: %v", req["format"], req["format"])
	}
	opts, _ := req["options"].(map[string]any)
	if opts["temperature"] != 0.0 {
		t.Fatalf("temperature = %v, want 0", opts["temperature"])
	}
}

func TestOllamaGenerateWithoutSchemaOmitsFormat(t *testing.T) {
	t.Parallel()
	srv, got := fakeOllama(t, http.StatusOK, `{"response":"plain text"}`)

	out, err := NewOllama(srv.URL, "").Generate(context.Background(), "hello", "")
	if err != nil || string(out) != "plain text" {
		t.Fatalf("out=%s err=%v", out, err)
	}
	if _, present := (*got)["format"]; present {
		t.Fatalf("format should be omitted, request = %+v", *got)
	}
	if (*got)["model"] != DefaultModel {
		t.Fatalf("model = %v, want default %s", (*got)["model"], DefaultModel)
	}
}

func TestOllamaGenerateErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		status  int
		body    string
		schema  string
		wantErr string
	}{
		{"http error", http.StatusInternalServerError, `{"error":"boom"}`, "", "status 500"},
		{"bad json", http.StatusOK, `not json`, "", "decode"},
		{"invalid schema", http.StatusOK, `{"response":"x"}`, `{not json`, "schema"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv, _ := fakeOllama(t, tc.status, tc.body)
			_, err := NewOllama(srv.URL, "m").Generate(context.Background(), "p", tc.schema)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestOllamaUnreachable(t *testing.T) {
	t.Parallel()
	_, err := NewOllama("http://127.0.0.1:1", "m").Generate(context.Background(), "p", "")
	if err == nil || !strings.Contains(err.Error(), "ollama request") {
		t.Fatalf("err = %v", err)
	}
}
