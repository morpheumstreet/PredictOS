package prompts

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Bookmaker builds aggregation prompts from generic analysis maps.
func Bookmaker(analyses []map[string]any, x402 []map[string]any, eventIdentifier, pmType string) (systemPrompt, userPrompt string) {
	systemPrompt = `You are a senior financial analyst who specializes in synthesizing multiple expert opinions on prediction markets.
Your task is to combine and consolidate analyses from multiple AI agents and external data sources into a single, authoritative assessment.
You weigh each agent's analysis based on their confidence levels and the consistency of their reasoning.
When agents disagree, you provide balanced perspective on both sides before making a final recommendation.
Your output is ALWAYS in JSON format and you are VERY STRICT about it.`

	var b strings.Builder
	b.WriteString("# Task: Aggregate Multiple Data Sources\n\n")
	b.WriteString(fmt.Sprintf("Event: %s (%s)\n\n## Agent analyses\n", eventIdentifier, pmType))
	if len(analyses) == 0 {
		b.WriteString("(No AI agent analyses provided)\n")
	} else {
		raw, _ := json.MarshalIndent(analyses, "", "  ")
		b.Write(raw)
		b.WriteByte('\n')
	}
	if len(x402) > 0 {
		b.WriteString("\n## External x402 sources\n")
		raw, _ := json.MarshalIndent(x402, "", "  ")
		b.Write(raw)
		b.WriteByte('\n')
	}
	b.WriteString(`
## Output JSON schema
Return consolidated JSON with fields: event_ticker, ticker, title, marketProbability, estimatedActualProbability, alphaOpportunity, hasAlpha, predictedWinner, winnerConfidence, recommendedAction, reasoning, confidence, keyFactors[], risks[], questionAnswer, analysisSummary, consensusSummary, disagreementNotes.
`)
	userPrompt = b.String()
	return systemPrompt, userPrompt
}
