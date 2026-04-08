package llm

import (
	"fmt"
	"strings"
)

// ExtractTextFromResponsesAPI parses OpenAI/Grok-style /v1/responses JSON for concatenated output_text.
func ExtractTextFromResponsesAPI(raw map[string]any) (model string, text string, totalTokens *int, err error) {
	if m, ok := raw["model"].(string); ok {
		model = m
	}
	out, ok := raw["output"].([]any)
	if !ok {
		return model, "", nil, fmt.Errorf("responses API: missing output array")
	}
	var b strings.Builder
	for _, item := range out {
		om, ok := item.(map[string]any)
		if !ok || om["type"] != "message" {
			continue
		}
		parts, _ := om["content"].([]any)
		for _, p := range parts {
			pm, ok := p.(map[string]any)
			if !ok {
				continue
			}
			if pm["type"] == "output_text" {
				if t, ok := pm["text"].(string); ok {
					b.WriteString(t)
					b.WriteByte('\n')
				}
			}
		}
	}
	if u, ok := raw["usage"].(map[string]any); ok {
		if v, ok := u["total_tokens"].(float64); ok {
			t := int(v)
			totalTokens = &t
		}
	}
	return model, strings.TrimSpace(b.String()), totalTokens, nil
}
