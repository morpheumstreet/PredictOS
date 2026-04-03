# Alpha Hunter: Dome-Native EPL Intelligence Engine

> A multi-agent AI system for autonomous sports prediction market trading, built for the PredictOS Hackathon.

[![Dome API](https://img.shields.io/badge/Powered%20by-Dome%20API-blue)](https://getdomeapi.io)
[![PredictOS](https://img.shields.io/badge/Built%20for-PredictOS-green)](https://github.com/PredictionXBT/PredictOS)

## Overview

Alpha Hunter is a **Dome-Native Intelligence Engine** that demonstrates deep integration with the Dome API for Polymarket sports betting. Unlike simple API wrappers, Alpha Hunter treats Dome as the **primary intelligence layer**, using real-time market signals to guide agent reasoning and execution.

### Key Innovation: The Dome-First Pipeline

```
┌─────────────────────────────────────────────────────────────────┐
│  Stage 1: DOME FUNNEL (System-Level, No LLM)                    │
│  ├── Fetch 100+ markets via Dome API                            │
│  ├── Calculate opportunity_score = volatility × log(volume)     │
│  │                                  × (1 - consensus)           │
│  └── Rank and select top 25 markets by opportunity              │
├─────────────────────────────────────────────────────────────────┤
│  Stage 2: AGENT REASONING (LLM-Level)                           │
│  ├── Pass ranked markets + FPL data to agents                   │
│  ├── Agents output: probability, rationale, uncertainty         │
│  └── Multi-model comparison (Llama 70B vs 8B)                   │
├─────────────────────────────────────────────────────────────────┤
│  Stage 3: EXECUTION & SIZING                                    │
│  ├── final_edge = raw_edge × plausibility × info_gain           │
│  │               × liquidity × volatility                       │
│  ├── Apply diversity quotas and conviction rules                │
│  └── Execute with Dome-aware position sizing                    │
└─────────────────────────────────────────────────────────────────┘
```

## Features

### Dome API Deep Integration
- **Opportunity Scoring**: Markets ranked by `volatility × log(volume) × (1 - consensus)`
- **Liquidity-Aware Sizing**: Trade size scales with market depth from Dome
- **Market Efficiency Scoring**: Real-time measurement of market quality
- **Dynamic Discovery**: No hardcoded markets—Dome tells agents where alpha exists

### Multi-Agent Intelligence
- **Llama 3.3 70B**: Deep analysis agent for complex reasoning
- **Llama 3.1 8B**: Fast agent for rapid market scanning
- **Cross-Agent Contention**: Automatic detection of meaningful disagreements
- **Consensus Metrics**: Track when agents agree vs. diverge

### Professional Risk Management
- **Information Gain Prior**: `info_gain = 1 - |prob - 0.5| × 2` — Penalizes "boring certainty"
- **Plausibility Factor**: `plausibility = clamp(0.5 + (market_price - 0.5), 0.25, 1.0)` — Dome-native sanity check
- **Title Edge Cap**: Forces agents to shine on props, not trivial shorts
- **Near-Miss Logging**: Shows markets that almost qualified (research-grade UX)
- **Explicit Abstention**: Agents explain why they chose not to trade

### Trade Style Classification
- **CONSENSUS**: Agent agrees with market direction
- **CONTRARIAN**: Agent disagrees >30% with market price

## Installation

```bash
# Clone the repository
git clone https://github.com/PredictionXBT/PredictOS.git
cd PredictOS/examples/alpha-hunter

# Install dependencies
pip install requests pandas groq

# Set your API keys in main.py or as environment variables
DOME_API_KEY = "your_dome_api_key"
GROQ_API_KEY = "your_groq_api_key"
```

## Usage

### Quick Start (Windows)
```bash
# Double-click or run:
run_alpha_hunter.bat
```

### Quick Start (Linux/Mac)
```bash
export PYTHONPATH=$PYTHONPATH:.
python3 main.py
```

## Output Example

```json
{
  "cycle_id": "20260202T154717Z",
  "market_efficiency": 96.16,
  "market_count": 25,
  "agent_results": {
    "Agent_Llama_70B": {
      "trades_executed": 2,
      "details": [
        {
          "market_slug": "will-west-ham-be-relegated...",
          "market_type": "RELEGATION",
          "trade_style": "CONSENSUS",
          "side": "buy_no",
          "adjusted_edge": 0.049,
          "information_gain_factor": 0.44,
          "rationale": "West Ham has average strength and low form..."
        }
      ],
      "near_miss_markets": [
        {"market_slug": "will-arsenal-win...", "adjusted_edge": 0.016}
      ]
    }
  },
  "contention_events": [
    {
      "market_slug": "will-west-ham-be-relegated...",
      "consensus_gap": 0.38,
      "agents": ["Agent_Llama_70B", "Agent_Llama_8B"],
      "probabilities": [0.22, 0.60]
    }
  ]
}
```

## Architecture

```
alpha_hunter/
├── main.py              # Orchestrator with contention detection
├── data_engine.py       # Dome API + FPL data integration
├── execution.py         # Risk management and trade execution
├── agents/
│   └── base_agent.py    # Multi-model agent framework
├── utils/
│   └── mapping.py       # Team/market mapping utilities
├── logs/                # Cycle results and execution logs
└── run_alpha_hunter.bat # Windows quick-start script
```

## Why This Wins: Dome API Prize Track

### 1. Deep Dome Integration (Not Just Fetching)
```python
# We don't just fetch markets—we REASON over them
opportunity_score = volatility * math.log(volume + 1) * (1 - consensus)
```

### 2. Dome Drives Agent Focus
Agents only see the top 25 markets by opportunity score. Dome decides what's worth analyzing.

### 3. Liquidity-Aware Execution
```python
liquidity_factor = 1.0 + (volume_24h / 1_000_000) * 0.5  # Scale with depth
final_size = base_size * liquidity_factor
```

### 4. Market Efficiency Scoring
```python
market_efficiency = 100 * (1 - avg_mispricing)  # Real-time quality metric
```

## Prize Track Alignment

| Track | Feature | Status |
|-------|---------|--------|
| **Dome API ($500)** | Opportunity scoring, liquidity factors, efficiency metrics | ✅ Implemented |
| **x402/PayAI ($500-$1000)** | Modular architecture ready for x402 tool payments | 🔧 Ready |
| **Privy ($500)** | Trade payloads ready for wallet signing | 🔧 Ready |

## Demo Highlights

1. **Watch agents discover markets via Dome** — No hardcoded slugs
2. **See contention events** — Agents disagree on West Ham relegation (Gap: 38%)
3. **Observe near-miss logging** — Arsenal title market almost qualified
4. **Track trade styles** — CONSENSUS vs CONTRARIAN classification

## Contributing

This module is designed to be merged into PredictOS as an example of Dome-native sports betting intelligence. PRs welcome!

## License

MIT License - See LICENSE file for details.

---

**Built for the PredictOS Hackathon 2026**  
*Powered by Dome API • Groq LLMs • Fantasy Premier League Data*
