import logging
import json
import time
import sys
import os
from datetime import datetime

# Setup logging to match main.py exactly
def setup_logging():
    if sys.platform == "win32":
        import io
        sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8')
        sys.stderr = io.TextIOWrapper(sys.stderr.buffer, encoding='utf-8')

    os.makedirs('logs', exist_ok=True)
    # Using a slightly different format to ensure it looks identical in terminal
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(levelname)s - %(message)s',
        handlers=[
            logging.StreamHandler(sys.stdout)
        ]
    )

class MockDataEngine:
    def get_alpha_bundle(self):
        return {
            "fpl_stats": {"West Ham": {"form": 2.0}, "Arsenal": {"form": 8.0}},
            "available_markets": [
                {
                    "slug": "will-west-ham-be-relegated-from-the-english-premier-league-after-the-202526-season",
                    "title": "Will West Ham be relegated?",
                    "price": 0.30, # Market implies 30%
                    "market_type": "RELEGATION",
                    "volume": 150000,
                    "volatility": 0.8,
                    "opportunity_score": 95.0
                },
                {
                    "slug": "will-arsenal-win-the-202526-english-premier-league",
                    "title": "Will Arsenal win the league?",
                    "price": 0.45,
                    "market_type": "WINNER",
                    "volume": 500000,
                    "volatility": 0.5,
                    "opportunity_score": 88.0
                },
                {
                    "slug": "will-leeds-win-the-202526-english-premier-league",
                    "title": "Will Leeds win the league?",
                    "price": 0.05,
                    "market_type": "WINNER",
                    "volume": 10000,
                    "volatility": 0.2,
                    "opportunity_score": 10.0
                }
            ]
        }

class MockAgent:
    def __init__(self, name):
        self.name = name

    def analyze_and_decide(self, bundle):
        time.sleep(2) # Simulate thinking
        if self.name == "Agent_Llama_70B":
            return {
                "evaluations": [
                    {
                        "market_slug": "will-west-ham-be-relegated-from-the-english-premier-league-after-the-202526-season",
                        "estimated_probability": 0.22, # Disagrees with market (0.30) slightly, but big disagreement with 8B
                        "rationale": "West Ham has average strength and low form..."
                    },
                    {
                        "market_slug": "will-arsenal-win-the-202526-english-premier-league",
                        "estimated_probability": 0.466, # Small edge (0.016) -> Near miss
                        "rationale": "Arsenal strong but price is efficient."
                    }
                ]
            }
        else: # 8B
            return {
                "evaluations": [
                    {
                        "market_slug": "will-west-ham-be-relegated-from-the-english-premier-league-after-the-202526-season",
                        "estimated_probability": 0.60, # Massive disagreement (gap 0.38 with 70B)
                        "rationale": "Chaos mode initiated."
                    }
                ]
            }

def run_demo():
    setup_logging()
    cycle_id = datetime.utcnow().strftime('%Y%m%dT%H%M%SZ')
    logging.info(f"STARTING Dome-Native Alpha Engine Cycle: {cycle_id}")
    time.sleep(1)

    logging.info("FETCHING FPL and Ranked Polymarket data (Stage 1: Funnel)...")
    time.sleep(2)
    
    # Simulate partial logs from DataEngine
    logging.info("DOME Market Efficiency Score: 96.16%")
    
    # Initialize mocks
    agents = [MockAgent("Agent_Llama_70B"), MockAgent("Agent_Llama_8B")]
    market_probs = {}

    # MAIN LOOP
    for agent in agents:
        logging.info(f"AGENT {agent.name} is analyzing (Stage 2: Reasoning)...")
        time.sleep(1.5)
        logging.info("HTTP Request: POST https://api.groq.com/openai/v1/chat/completions \"HTTP/1.1 200 OK\"")
        
        decision = agent.analyze_and_decide(None)
        
        # Simulate Execution Engine logs
        if agent.name == "Agent_Llama_70B":
            # West Ham: Price 0.30, Prob 0.22. Diff 0.08. 
            # Arsenal: Price 0.45, Prob 0.466. Diff 0.016.
            time.sleep(0.5)
            logging.info("GATE: Agent_Llama_70B | will-leeds-win-the-202526-english-premie... | InfoGain 0.10 < 0.4 | SKIPPED (boring certainty)")
            time.sleep(0.2)
            logging.info("GATE: Agent_Llama_70B | will-arsenal-win-the-202526-english-prem... | TITLE edge 0.016 < 0.05 | SKIPPED (low edge)")
            time.sleep(0.2)
            logging.info("NEAR-MISS: Agent_Llama_70B | 1 markets almost qualified")
            
            # Record probs for contention
            market_probs["west-ham"] = market_probs.get("west-ham", []) + [(agent.name, 0.22)]

        else: # 8B
            # West Ham: Price 0.30, Prob 0.60.
            time.sleep(0.5)
            logging.info("GATE: Agent_Llama_8B | will-leeds-win-the-202526-english-premie... | InfoGain 0.10 < 0.4 | SKIPPED (boring certainty)")
            time.sleep(0.2)
            logging.info("EXECUTE: Agent_Llama_8B | CONTRARIAN | RELEGATION | will-west-ham-be-relegated... | buy_yes | Edge: 0.30 | InfoGain: 0.58 | Size: $15.0")
            
            # Record probs
            market_probs["west-ham"] = market_probs.get("west-ham", []) + [(agent.name, 0.60)]

    # CONTENTION CHECK
    time.sleep(1)
    # West Ham: 0.22 vs 0.60 -> Gap 0.38
    logging.info("CONTENTION: High disagreement on will-west-ham-be-relegated... (Gap: 0.38)")
    
    # Finalize
    time.sleep(1)
    result_path = f"logs/cycle_{cycle_id}.json"
    logging.info(f"CYCLE COMPLETE. Results saved to {result_path}")

    # Generate the dummy JSON file so the user can show it (Scene 6)
    dummy_json = {
        "cycle_id": cycle_id,
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
    
    with open("logs/latest_results.json", "w") as f:
        json.dump(dummy_json, f, indent=2)

if __name__ == "__main__":
    run_demo()
