// Package llm talks to large language models behind a small Provider interface.
//
// Callers (explain, fix) depend on Provider, not on a concrete backend, so tests
// can use a fake and new backends can be added without touching callers.
package llm

import "context"

// Provider generates a completion for a prompt.
//
// When schema is a non-empty JSON Schema, the backend must constrain its output
// to JSON matching that schema. An empty schema means free-form text.
type Provider interface {
	Generate(ctx context.Context, prompt, schema string) ([]byte, error)
}
