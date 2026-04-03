import os
import json
import logging
from datetime import datetime
from data_engine import DataEngine
from agents.base_agent import get_agents
from execution import ExecutionEngine

# Configuration
# Configuration
DOME_API_KEY = os.getenv("DOME_API_KEY", "your_dome_api_key")
GROQ_API_KEY = os.getenv("GROQ_API_KEY", "your_groq_api_key")

def setup_logging():
    import sys
    if sys.platform == "win32":
        import io
        sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8')
        sys.stderr = io.TextIOWrapper(sys.stderr.buffer, encoding='utf-8')

    os.makedirs('logs', exist_ok=True)
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(levelname)s - %(message)s',
        handlers=[
            logging.FileHandler("logs/execution.log", encoding='utf-8'),
            logging.StreamHandler(sys.stdout)
        ]
    )

def calculate_market_efficiency(markets):
    """Manus Bonus: Market Efficiency Score."""
    if not markets: return 100.0
    avg_opp = sum(m.get('opportunity_score', 0) for m in markets) / len(markets)
    efficiency = max(0, 100 - (avg_opp * 20))
    return round(efficiency, 2)

def run_trading_cycle(dome_key, groq_key):
    cycle_id = datetime.utcnow().strftime('%Y%m%dT%H%M%SZ')
    logging.info(f"STARTING Dome-Native Alpha Engine Cycle: {cycle_id}")
    
    engine = DataEngine(dome_key)
    executor = ExecutionEngine(max_budget=100.0)
    
    logging.info("FETCHING FPL and Ranked Polymarket data (Stage 1: Funnel)...")
    bundle = engine.get_alpha_bundle()
    
    efficiency_score = calculate_market_efficiency(bundle['available_markets'])
    logging.info(f"DOME Market Efficiency Score: {efficiency_score}%")
    
    agents = get_agents(groq_key)
    cycle_results = {
        "cycle_id": cycle_id,
        "timestamp": datetime.utcnow().timestamp(),
        "market_efficiency": efficiency_score,
        "market_count": len(bundle['available_markets']),
        "agent_results": {},
        "contention_events": []
    }
    
    all_evals = {}
    
    for agent in agents:
        logging.info(f"AGENT {agent.name} is analyzing (Stage 2: Reasoning)...")
        decision = agent.analyze_and_decide(bundle)
        evals = decision.get('evaluations', [])
        all_evals[agent.name] = evals
        
        # Stage 3: Execution & Sizing (now returns dict with trades, abstention, near_miss)
        result = executor.process_evaluations(agent.name, evals, bundle['available_markets'])
        
        executed_trades = result.get('trades', [])
        abstention_reason = result.get('abstention_reason')
        near_miss_markets = result.get('near_miss_markets', [])
        
        agent_result = {
            "evaluations_count": len(evals),
            "trades_executed": len(executed_trades),
            "details": executed_trades
        }
        
        # TUNE #1: Add abstention reason if agent abstained
        if abstention_reason:
            agent_result["abstention_reason"] = abstention_reason
        
        # TUNE #2: Add near-miss markets (top 2 only, minimal fields)
        if near_miss_markets:
            agent_result["near_miss_markets"] = near_miss_markets
        
        cycle_results["agent_results"][agent.name] = agent_result

    # Upgrade #3: Cross-Agent Contention
    market_probs = {}
    for agent_name, evals in all_evals.items():
        for ev in evals:
            slug = ev.get('market_slug')
            prob = ev.get('estimated_probability', 0.5)
            if slug not in market_probs: market_probs[slug] = []
            market_probs[slug].append((agent_name, prob))

    for slug, probs in market_probs.items():
        if len(probs) > 1:
            p_values = [p[1] for p in probs]
            gap = max(p_values) - min(p_values)
            if gap > 0.2: # Significant disagreement
                cycle_results["contention_events"].append({
                    "market_slug": slug,
                    "consensus_gap": round(gap, 3),
                    "agents": [p[0] for p in probs],
                    "probabilities": [p[1] for p in probs]
                })
                logging.info(f"CONTENTION: High disagreement on {slug} (Gap: {round(gap, 3)})")

    # Save results
    result_path = f"logs/cycle_{cycle_id}.json"
    with open(result_path, 'w', encoding='utf-8') as f:
        json.dump(cycle_results, f, indent=2)
    
    with open("logs/latest_results.json", 'w', encoding='utf-8') as f:
        json.dump(cycle_results, f, indent=2)
        
    logging.info(f"CYCLE COMPLETE. Results saved to {result_path}")

if __name__ == "__main__":
    setup_logging()
    run_trading_cycle(DOME_API_KEY, GROQ_API_KEY)
