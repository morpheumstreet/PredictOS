import logging

class ExecutionEngine:
    def __init__(self, max_budget=100.0):
        self.max_budget = max_budget
        self.agent_spend = {}
        self.market_weights = {"PLAYER_PROP": 1.0, "MANAGER": 1.0, "TOP_4": 0.8, "RELEGATION": 0.8, "PLACEMENT": 0.5, "TITLE": 0.3}

    def process_evaluations(self, agent_name, evaluations, available_markets):
        if agent_name not in self.agent_spend:
            self.agent_spend[agent_name] = 0.0
        
        # Build EXACT slug map
        market_map = {m['slug']: m for m in available_markets if m.get('slug')}
        
        processed_evals = []
        near_miss_candidates = []  # Track markets that almost qualified
        
        for ev in evaluations:
            slug = ev.get('market_slug', '')
            
            # EXACT MATCH ONLY - no fuzzy matching
            if slug not in market_map:
                logging.warning(f"REJECTED: {agent_name} tried to use invalid slug: {slug}")
                continue
            
            market = market_map[slug]
            prob = ev.get('estimated_probability', 0.5)
            market_price = market.get('price', 0.5)
            disagreement = abs(prob - market_price)
            side = "buy_yes" if prob > market_price else "buy_no"
            
            # Trade Style Labeling
            trade_style = "CONTRARIAN" if disagreement > 0.3 else "CONSENSUS"
            
            m_type = market.get('market_type', 'PLACEMENT')
            weight = self.market_weights.get(m_type, 0.5)
            raw_edge = disagreement * weight
            
            # PLAUSIBILITY PRIOR (Dome-native sanity check)
            plausibility_factor = max(0.25, min(1.0, 0.5 + (market_price - 0.5)))
            
            # INFORMATION GAIN PRIOR (Domain-agnostic learning value)
            information_gain_factor = 1 - abs(prob - 0.5) * 2
            information_gain_factor = max(0.1, information_gain_factor)  # Floor at 0.1
            
            # Liquidity & Volatility factors
            vol = market.get('volatility', 0.05)
            vol_factor = 1.0 + (vol * 5)
            volume = market.get('volume_total', 1000)
            liq_factor = 1.0 + (min(volume, 1000000) / 1000000) * 0.5
            
            # FINAL EDGE: All factors combined
            adjusted_edge = raw_edge * plausibility_factor * information_gain_factor
            
            # Build the evaluation record
            eval_record = {
                "market_slug": slug,
                "market_type": m_type,
                "trade_style": trade_style,
                "side": side,
                "prob": prob,
                "market_price": market_price,
                "disagreement": round(disagreement, 3),
                "raw_edge": round(raw_edge, 3),
                "plausibility_factor": round(plausibility_factor, 2),
                "information_gain_factor": round(information_gain_factor, 2),
                "adjusted_edge": round(adjusted_edge, 3),
                "liquidity_factor": round(liq_factor, 2),
                "volatility_factor": round(vol_factor, 2),
                "size_usd": 0,  # Will be set later
                "rationale": ev.get('rationale', '')
            }
            
            # ============================================================
            # GATE #1: MINIMUM INFORMATION GAIN GATE
            # ============================================================
            if information_gain_factor < 0.4:
                logging.info(f"GATE: {agent_name} | {slug[:40]}... | InfoGain {information_gain_factor:.2f} < 0.4 | SKIPPED (boring certainty)")
                # Track as near-miss if it passed plausibility
                if plausibility_factor >= 0.25:
                    near_miss_candidates.append({
                        "market_slug": slug,
                        "adjusted_edge": round(adjusted_edge, 3),
                        "information_gain_factor": round(information_gain_factor, 2),
                        "gate_failed": "information_gain"
                    })
                continue
            
            # ============================================================
            # GATE #2: TITLE MARKET EDGE CAP
            # ============================================================
            if m_type == "TITLE" and adjusted_edge < 0.05:
                logging.info(f"GATE: {agent_name} | {slug[:40]}... | TITLE edge {adjusted_edge:.3f} < 0.05 | SKIPPED (low edge)")
                # Track as near-miss
                near_miss_candidates.append({
                    "market_slug": slug,
                    "adjusted_edge": round(adjusted_edge, 3),
                    "information_gain_factor": round(information_gain_factor, 2),
                    "gate_failed": "title_edge"
                })
                continue
            
            # Calculate size for valid trades
            size = round(5.0 * adjusted_edge * liq_factor * vol_factor, 2)
            size = max(2.0, min(10.0, size))
            eval_record['size_usd'] = size
            
            processed_evals.append(eval_record)
        
        # Sort by adjusted edge
        processed_evals.sort(key=lambda x: x['adjusted_edge'], reverse=True)
        
        # Apply diversity quotas: Max 1 TITLE, Max 2 of others
        selected_trades = []
        type_counts = {}
        for trade in processed_evals:
            m_type = trade['market_type']
            if m_type == "TITLE" and type_counts.get(m_type, 0) >= 1:
                continue
            if type_counts.get(m_type, 0) >= 2:
                continue
            selected_trades.append(trade)
            type_counts[m_type] = type_counts.get(m_type, 0) + 1
            if len(selected_trades) >= 4:
                break
        
        # Conviction Rule: Ensure at least one YES (per-agent)
        if selected_trades and all(t['side'] == "buy_no" for t in selected_trades):
            best_yes = next((t for t in processed_evals if t['side'] == "buy_yes"), None)
            if best_yes:
                selected_trades[-1] = best_yes
                logging.info(f"CONVICTION: {agent_name} forced YES on {best_yes['market_slug'][:40]}...")
        
        # Execute trades
        executed_trades = []
        for trade in selected_trades:
            size = trade['size_usd']
            if self.agent_spend[agent_name] + size <= self.max_budget:
                self.agent_spend[agent_name] += size
                executed_trades.append(trade)
                logging.info(f"EXECUTE: {agent_name} | {trade['trade_style']} | {trade['market_type']} | {trade['market_slug'][:50]} | {trade['side']} | Edge: {trade['adjusted_edge']} | InfoGain: {trade['information_gain_factor']} | Size: ${size}")
        
        # ============================================================
        # TUNE #1: EXPLICIT ABSTENTION REASON
        # ============================================================
        abstention_reason = None
        if not executed_trades:
            abstention_reason = "No markets cleared information_gain >= 0.4 and adjusted_edge >= 0.03"
            logging.info(f"ABSTAIN: {agent_name} | {abstention_reason}")
        
        # ============================================================
        # TUNE #2: NEAR-MISS MARKETS (top 2 only)
        # Sort by adjusted_edge DESC, take top 2
        # ============================================================
        near_miss_candidates.sort(key=lambda x: x['adjusted_edge'], reverse=True)
        near_miss_markets = near_miss_candidates[:2]
        
        # Clean near-miss output (only slug, edge, info_gain)
        near_miss_clean = [
            {
                "market_slug": nm['market_slug'],
                "adjusted_edge": nm['adjusted_edge'],
                "information_gain_factor": nm['information_gain_factor']
            }
            for nm in near_miss_markets
        ]
        
        if near_miss_clean:
            logging.info(f"NEAR-MISS: {agent_name} | {len(near_miss_clean)} markets almost qualified")
        
        return {
            "trades": executed_trades,
            "abstention_reason": abstention_reason,
            "near_miss_markets": near_miss_clean
        }
