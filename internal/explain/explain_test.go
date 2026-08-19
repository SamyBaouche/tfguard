package explain

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func sampleInput() Input {
	return Input{
		MaxRisk:      "CRITICAL",
		ChangeCount:  1,
		Creates:      1,
		Updates:      1,
		Deletes:      1,
		CostDeltaUSD: 7.59,
		Changes: []ChangeLine{{
			Address: "aws_db_instance.main",
			Type:    "aws_db_instance",
			Action:  "delete",
			Risk:    "CRITICAL",
		}},
		Findings: []FindingLine{{
			ID:       "TFGUARD-RDS-001",
			Severity: "HIGH",
			Resource: "aws_db_instance.main",
			Title:    "unencrypted",
		}},
	}
}

func TestInputJSON(t *testing.T) {
	t.Parallel()
	in := sampleInput()
	b, err := InputJSON(in)
	if err != nil || !strings.Contains(string(b), "CRITICAL") {
		t.Fatalf("json = %s err=%v", b, err)
	}
}

func TestParseExplanation(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"summary":"Risky plan","risks":["delete db"],"recommendations":["review"],"cost_note":"+7 USD/mo"}`)
	got, err := parseExplanation(raw)
	if err != nil || got.Summary != "Risky plan" {
		t.Fatalf("parse: %v got=%+v", err, got)
	}
}

func TestParseExplanationWrappedJSON(t *testing.T) {
	t.Parallel()
	raw := []byte("Here is the JSON:\n{\"summary\":\"ok\",\"risks\":[],\"recommendations\":[],\"cost_note\":\"none\"}\n")
	got, err := parseExplanation(raw)
	if err != nil || got.Summary != "ok" {
		t.Fatalf("parse wrapped: %v", err)
	}
}

func TestExplainSkip(t *testing.T) {
	t.Parallel()
	got, err := Explain(context.Background(), sampleInput(), Options{Skip: true})
	if err != nil || !got.Skipped || got.Explanation != nil {
		t.Fatalf("skip = %+v err=%v", got, err)
	}
}

func TestExplainCacheHit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	in := sampleInput()
	ctxJSON, err := InputJSON(in)
	if err != nil {
		t.Fatal(err)
	}
	path, err := cachePath(dir, ctxJSON)
	if err != nil {
		t.Fatal(err)
	}
	cached := Explanation{
		Summary:         "cached summary",
		Risks:           []string{"cached risk"},
		Recommendations: []string{"cached rec"},
		CostNote:        "cached cost",
	}
	if err := saveCache(path, cached); err != nil {
		t.Fatal(err)
	}

	got, err := Explain(context.Background(), in, Options{CacheDir: dir})
	if err != nil || !got.Cached || got.Explanation.Summary != "cached summary" {
		t.Fatalf("cache hit = %+v err=%v", got, err)
	}
}

func TestExplainOllama(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"response": `{"summary":"Test summary","risks":["delete RDS"],"recommendations":["block deploy"],"cost_note":"small increase"}`,
		})
	}))
	defer srv.Close()

	dir := t.TempDir()
	in := sampleInput()
	got, err := Explain(context.Background(), in, Options{
		OllamaURL: srv.URL,
		Model:     "test",
		CacheDir:  dir,
	})
	if err != nil || got.Explanation == nil || got.Explanation.Summary != "Test summary" {
		t.Fatalf("ollama = %+v err=%v", got, err)
	}

	got2, err := Explain(context.Background(), in, Options{
		OllamaURL: srv.URL,
		CacheDir:  dir,
	})
	if err != nil || !got2.Cached {
		t.Fatalf("expected cache hit, got %+v", got2)
	}
}

func TestExplainOllamaUnavailable(t *testing.T) {
	t.Parallel()
	got, err := Explain(context.Background(), sampleInput(), Options{
		OllamaURL: "http://127.0.0.1:1",
		CacheDir:  t.TempDir(),
	})
	if err != nil || got.Explanation != nil || got.Warning == "" {
		t.Fatalf("expected soft warning, got %+v err=%v", got, err)
	}
}

func TestCachePathDeterministic(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	a, err := cachePath(dir, []byte(`{"x":1}`))
	if err != nil {
		t.Fatal(err)
	}
	b, err := cachePath(dir, []byte(`{"x":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(a) != filepath.Base(b) || !strings.HasSuffix(a, ".json") {
		t.Fatalf("paths = %s %s", a, b)
	}
}
