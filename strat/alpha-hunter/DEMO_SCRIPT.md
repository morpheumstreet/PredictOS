# Alpha Hunter Demo Script & X Post Template

## X Post Template (Copy & Paste)

### Main Post (Thread Starter)
```
🎯 Introducing Alpha Hunter: A Dome-Native Intelligence Engine for @Polymarket

Built for the @prediction_xbt Hackathon

What makes it special? Dome isn't just a data source—it's the BRAIN.

🧵 Thread below 👇

@getdomeapi @CoinbaseDev @PayAINetwork
```

### Thread Post 2
```
1/ THE DOME-FIRST PIPELINE

Most bots: "Fetch data → Trade"
Alpha Hunter: "Let Dome RANK opportunities → Then reason"

We calculate an Opportunity Score:
📊 volatility × log(volume) × (1 - consensus)

Dome tells agents WHERE alpha exists.
```

### Thread Post 3
```
2/ MULTI-AGENT CONTENTION

We run TWO agents (Llama 70B + 8B) on the same markets.

When they DISAGREE, we log it as a "Contention Event"

Example: West Ham Relegation
• 70B: 22% probability
• 8B: 60% probability
• Gap: 38% 🔥

This is emergent intelligence.
```

### Thread Post 4
```
3/ INFORMATION GAIN PRIOR

We penalize "boring certainty" trades.

If an agent says "Leeds won't win the league" (prob = 0.01)...
→ Information Gain = 0.1 (heavily penalized)

We want trades that TEACH us something, not confirm the obvious.
```

### Thread Post 5
```
4/ NEAR-MISS MARKETS

Research-grade UX: We show markets that ALMOST qualified.

"Arsenal title market: adjusted_edge = 0.016, info_gain = 0.3"

This proves our thresholds are intentional, not arbitrary.

Judges love transparency.
```

### Thread Post 6 (Final)
```
5/ THE RESULT

✅ Dome-native opportunity scoring
✅ Multi-agent contention detection
✅ Information gain priors
✅ Liquidity-aware sizing
✅ Near-miss logging

Full code: [GitHub PR Link]

Built for @prediction_xbt Hackathon
Powered by @getdomeapi 🚀

#PredictOS #Polymarket #AIAgents
```

---

## Video Demo Script (60-90 seconds)

### Scene 1: Introduction (10 sec)
**Voiceover**: "This is Alpha Hunter—a Dome-native intelligence engine for Polymarket sports betting."

**Screen**: Show the terminal with `run_alpha_hunter.bat` ready to execute.

### Scene 2: The Dome Funnel (15 sec)
**Voiceover**: "Stage 1: The Dome Funnel. We fetch 100+ markets and rank them by opportunity score—volatility times volume times uncertainty. Dome tells agents where alpha exists."

**Screen**: Show the log output with "FETCHING FPL and Ranked Polymarket data..."

### Scene 3: Agent Reasoning (20 sec)
**Voiceover**: "Stage 2: Agent Reasoning. We run two Llama models—70B for deep analysis, 8B for fast scanning. Watch them analyze the same markets with different conclusions."

**Screen**: Show the log output with "AGENT Agent_Llama_70B is analyzing..." and "AGENT Agent_Llama_8B is analyzing..."

### Scene 4: The Gates (15 sec)
**Voiceover**: "The system applies strict gates. Information gain below 0.4? Skipped. Title market with low edge? Skipped. We only execute high-quality trades."

**Screen**: Show the GATE log messages with "SKIPPED (boring certainty)"

### Scene 5: Contention Events (15 sec)
**Voiceover**: "The magic moment: Contention Events. When agents disagree by more than 20%, we flag it. West Ham relegation—70B says 22%, 8B says 60%. That's a 38% gap. This is emergent intelligence."

**Screen**: Show the CONTENTION log messages.

### Scene 6: Results (15 sec)
**Voiceover**: "The final output: structured JSON with trade details, near-miss markets, and contention events. Research-grade transparency."

**Screen**: Show the `latest_results.json` file with the clean output.

### Scene 7: Closing (10 sec)
**Voiceover**: "Alpha Hunter. Dome-native intelligence for prediction markets. Built for the PredictOS Hackathon."

**Screen**: Show the README with badges and GitHub link.

---

## Key Demo Moments to Capture

1. **The GATE messages** - Shows the system rejecting "boring" trades
2. **The EXECUTE messages** - Shows actual trades being logged
3. **The CONTENTION messages** - Shows agents disagreeing
4. **The near_miss_markets in JSON** - Shows research-grade UX
5. **The market_efficiency score** - Shows Dome-native metrics

## Recording Tips

1. Use a clean terminal with large font (16pt+)
2. Run the script fresh so all logs appear in sequence
3. Pause briefly on key moments (GATE, EXECUTE, CONTENTION)
4. Keep the video under 90 seconds for X engagement
5. Add captions for accessibility

---

## Hashtags for X Post
```
#PredictOS #Polymarket #AIAgents #DomeAPI #Hackathon #CryptoAI #PredictionMarkets #LLM #Groq
```
