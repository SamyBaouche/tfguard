// Package explain calls a local Ollama LLM to summarize a tfguard scan report.
package explain

import (
	"context"
	"fmt"
)

// Result holds an explanation or a soft skip/warning.
type Result struct {
	Explanation *Explanation
	Cached      bool
	Skipped     bool
	Warning     string
}

// Options configures the explainer.
type Options struct {
	Skip      bool   // --no-ai
	OllamaURL string // default http://127.0.0.1:11434
	Model     string // default llama3.2
	CacheDir  string // default $XDG_CACHE_HOME/tfguard/explain
}

// Explain builds context, checks cache, and calls Ollama when enabled.
func Explain(ctx context.Context, in Input, opts Options) (Result, error) {
	if opts.Skip {
		return Result{Skipped: true}, nil
	}

	ctxJSON, err := InputJSON(in)
	if err != nil {
		return Result{}, err
	}

	cacheFile, err := cachePath(opts.CacheDir, ctxJSON)
	if err != nil {
		return Result{}, err
	}
	if cached, ok, err := loadCache(cacheFile); err != nil {
		return Result{}, err
	} else if ok {
		exp := cached
		return Result{Explanation: &exp, Cached: true}, nil
	}

	client := newOllamaClient(opts.OllamaURL, opts.Model)
	raw, err := client.generate(ctx, buildPrompt(ctxJSON))
	if err != nil {
		return Result{Warning: fmt.Sprintf("AI explainer unavailable: %v", err)}, nil
	}

	exp, err := parseExplanation(raw)
	if err != nil {
		return Result{}, fmt.Errorf("invalid LLM response: %w", err)
	}
	if err := saveCache(cacheFile, exp); err != nil {
		return Result{}, err
	}
	return Result{Explanation: &exp}, nil
}
