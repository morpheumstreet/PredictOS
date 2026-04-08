package llm

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// Facade routes models to OpenAI, Grok, or BlockRun responses/chat APIs.
type Facade struct {
	http *http.Client
}

func NewFacade(hc *http.Client) *Facade {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &Facade{http: hc}
}

func IsOpenAIModel(model string) bool {
	m := strings.TrimSpace(model)
	if strings.HasPrefix(m, "gpt-") {
		return true
	}
	for _, p := range []string{"gpt-5.2", "gpt-5.1", "gpt-5-nano", "gpt-4.1", "gpt-4.1-mini", "o1", "o3"} {
		if strings.HasPrefix(m, p) {
			return true
		}
	}
	return false
}

// CompleteJSON returns concatenated model text (expected JSON) from the configured provider.
func (f *Facade) CompleteJSON(ctx context.Context, model, systemPrompt, userPrompt string, grokTools []string) (text string, responseModel string, totalTokens *int, paymentCost *string, err error) {
	switch {
	case isBlockRunModel(model):
		return callBlockRun(ctx, f.http, model, systemPrompt, userPrompt, true, grokHasSearch(grokTools))
	case IsOpenAIModel(model):
		raw, err := callOpenAIResponses(ctx, f.http, model, systemPrompt, userPrompt, "json_object")
		if err != nil {
			return "", "", nil, nil, err
		}
		rm, txt, tok, err := ExtractTextFromResponsesAPI(raw)
		return txt, rm, tok, nil, err
	default:
		raw, err := callGrokResponses(ctx, f.http, model, systemPrompt, userPrompt, "json_object", grokTools)
		if err != nil {
			return "", "", nil, nil, err
		}
		rm, txt, tok, err := ExtractTextFromResponsesAPI(raw)
		return txt, rm, tok, nil, err
	}
}

func grokHasSearch(tools []string) bool {
	for _, t := range tools {
		if t == "x_search" || t == "web_search" {
			return true
		}
	}
	return false
}

// Provider returns a label for logging.
func Provider(model string) string {
	switch {
	case isBlockRunModel(model):
		return "blockrun"
	case IsOpenAIModel(model):
		return "openai"
	default:
		return "grok"
	}
}

// CompleteJSONOrError wraps errors with provider context.
func (f *Facade) CompleteJSONOrError(ctx context.Context, model, system, user string, tools []string) (string, error) {
	text, _, _, _, err := f.CompleteJSON(ctx, model, system, user, tools)
	if err != nil {
		return "", fmt.Errorf("%s: %w", Provider(model), err)
	}
	return text, nil
}
