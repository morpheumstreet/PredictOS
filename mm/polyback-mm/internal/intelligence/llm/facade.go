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

// CompletionOpts optional tuning for a single completion. Nil fields mean “use provider default”
// (for temperature) or, for JSONObject, “true” when opts is nil (legacy CompleteJSON behavior).
type CompletionOpts struct {
	Temperature *float64
	// JSONObject requests json_object-style output when true. When false, plain text (no JSON schema hint at API level).
	JSONObject *bool
}

// CompleteJSON returns concatenated model text (expected JSON) from the configured provider.
func (f *Facade) CompleteJSON(ctx context.Context, model, systemPrompt, userPrompt string, grokTools []string) (text string, responseModel string, totalTokens *int, paymentCost *string, err error) {
	return f.CompleteJSONWithOptions(ctx, model, systemPrompt, userPrompt, grokTools, nil)
}

// CompleteJSONWithOptions is like CompleteJSON but allows temperature and toggling JSON response format
// (matches strat/alpha-rules description_agent.py OpenAI options).
func (f *Facade) CompleteJSONWithOptions(ctx context.Context, model, systemPrompt, userPrompt string, grokTools []string, opts *CompletionOpts) (text string, responseModel string, totalTokens *int, paymentCost *string, err error) {
	jsonObject := true
	var temperature *float64
	if opts != nil {
		temperature = opts.Temperature
		if opts.JSONObject != nil {
			jsonObject = *opts.JSONObject
		}
	}
	switch {
	case isBlockRunModel(model):
		return callBlockRun(ctx, f.http, model, systemPrompt, userPrompt, jsonObject, grokHasSearch(grokTools), temperature)
	case IsOpenAIModel(model):
		raw, err := callOpenAIResponses(ctx, f.http, model, systemPrompt, userPrompt, jsonObject, temperature)
		if err != nil {
			return "", "", nil, nil, err
		}
		rm, txt, tok, err := ExtractTextFromResponsesAPI(raw)
		return txt, rm, tok, nil, err
	default:
		raw, err := callGrokResponses(ctx, f.http, model, systemPrompt, userPrompt, jsonObject, grokTools, temperature)
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
	text, _, _, _, err := f.CompleteJSONWithOptions(ctx, model, system, user, tools, nil)
	if err != nil {
		return "", fmt.Errorf("%s: %w", Provider(model), err)
	}
	return text, nil
}
