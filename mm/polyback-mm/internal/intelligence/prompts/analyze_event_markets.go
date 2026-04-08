package prompts

import (
	"encoding/json"
	"fmt"
	"strings"
)

// AnalyzeEventMarkets builds system and user prompts (parity with TS analyzeEventMarketsPrompt).
func AnalyzeEventMarkets(markets []any, eventIdentifier, question, pmType string, tools []string, userCommand string) (systemPrompt, userPrompt string) {
	hasX := false
	hasWeb := false
	for _, t := range tools {
		if t == "x_search" {
			hasX = true
		}
		if t == "web_search" {
			hasWeb = true
		}
	}
	var toolInstr string
	if hasX {
		toolInstr += "\nYou have access to X (Twitter) search. Use it to find the latest posts, news, and sentiment about this event. When you find relevant posts that back your analysis, include their URLs in your response."
	}
	if hasWeb {
		toolInstr += "\nYou have access to web search. Use it to find the latest news articles, reports, and analysis about this event. When you find relevant resources that back your analysis, include their URLs in your response."
	}
	userCmdBlock := ""
	if strings.TrimSpace(userCommand) != "" {
		userCmdBlock = fmt.Sprintf(`
### VERY IMPORTANT: The user has provided specific commands that you MUST prioritize over everything else:
%s`, userCommand)
	}

	systemPrompt = fmt.Sprintf(`You are a financial analyst expert in the field of prediction markets (posted on %s) that understands the latest news, events, and market trends.
Your expertise lies in deeply analyzing prediction markets for a specific event, identifying if there's alpha (mispricing) opportunity, and providing a clear recommendation on which side (YES or NO) is more likely to win based on your analysis.
You always provide a short analysisSummary of your findings, less than 270 characters, that is very conversational and understandable by a non-expert who just wants to understand which side it might make more sense to buy into.
%s%s
Your output is ALWAYS in JSON format and you are VERY STRICT about it. You must return valid JSON that matches the exact schema specified.`, pmType, toolInstr, userCmdBlock)

	xSrc := ""
	if hasX {
		xSrc = `,
  "xSources": ["string"]`
	}
	webSrc := ""
	if hasWeb {
		webSrc = `,
  "webSources": ["string"]`
	}
	userExtra := ""
	if strings.TrimSpace(userCommand) != "" {
		userExtra = fmt.Sprintf(`
### VERY IMPORTANT:Below are the user commands - Make sure to prioritize their ask over everything else
%s
`, userCommand)
	}

	marketsJSON, _ := json.MarshalIndent(markets, "", "  ")
	userPrompt = fmt.Sprintf(`# Task: Deep Analysis of an Event's Prediction Markets

You are analyzing all markets for a specific event (%s) to determine:
1. Whether there is an alpha (mispricing) opportunity in any of the markets
2. Which market has the best alpha opportunity (if any)
3. Which side (YES or NO) is more likely to win for that market
4. Your confidence level in this assessment

## User's query/input/question About This Event
%s

## Platform: %s

## Event Markets (%d market(s))

%s

## Output Format

Return your analysis in JSON format with the following fields:

{
  "event_ticker": "string",
  "ticker": "string",
  "title": "string",
  "marketProbability": number,
  "estimatedActualProbability": number,
  "alphaOpportunity": number,
  "hasAlpha": boolean,
  "predictedWinner": "YES or NO",
  "winnerConfidence": number,
  "recommendedAction": "BUY YES | BUY NO | NO TRADE",
  "reasoning": "string",
  "confidence": number,
  "keyFactors": ["string"],
  "risks": ["string"],
  "questionAnswer": "string",
  "analysisSummary": "string"%s%s
}

Now analyze these markets and provide your assessment.%s`,
		eventIdentifier, question, pmType, len(markets), string(marketsJSON), xSrc, webSrc, userExtra)

	return systemPrompt, userPrompt
}
