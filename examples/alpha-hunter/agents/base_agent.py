import json
from groq import Groq

class AlphaAgent:
    def __init__(self, name, model_id, api_key):
        self.name = name
        self.model_id = model_id
        self.client = Groq(api_key=api_key)

    def analyze_and_decide(self, alpha_bundle):
        markets = alpha_bundle.get('available_markets', [])
        
        # CRITICAL: If no markets, return empty evaluations (no hallucination)
        if not markets:
            return {"evaluations": []}
        
        prompt = self._construct_prompt(alpha_bundle)
        try:
            response = self.client.chat.completions.create(
                model=self.model_id,
                messages=[
                    {"role": "system", "content": "You are an expert sports analyst. Return valid JSON only. You MUST use the EXACT market slugs provided."},
                    {"role": "user", "content": prompt}
                ],
                response_format={"type": "json_object"}
            )
            result = json.loads(response.choices[0].message.content)
            
            # VALIDATION: Filter out any hallucinated slugs
            valid_slugs = {m['slug'] for m in markets}
            valid_evals = [e for e in result.get('evaluations', []) if e.get('market_slug') in valid_slugs]
            
            return {"evaluations": valid_evals}
        except Exception as e:
            print(f"Error in Groq API call for {self.name}: {e}")
            return {"evaluations": []}

    def _construct_prompt(self, bundle):
        markets = bundle['available_markets']
        
        # Build explicit market list with numbered slugs
        market_list = ""
        for i, m in enumerate(markets, 1):
            market_list += f"{i}. SLUG: \"{m['slug']}\"\n   Title: {m['title']}\n   Current Price: {m['price']}\n   Type: {m['market_type']}\n\n"
        
        return f"""
Analyze the following English Premier League markets and estimate probabilities.

FPL Team Stats:
{json.dumps(bundle['fpl_stats'], indent=2)}

=== POLYMARKET MARKETS (from Dome API) ===
{market_list}

CRITICAL INSTRUCTIONS:
1. You MUST copy the SLUG exactly as shown above (including hyphens and numbers).
2. Example of correct slug: "will-arsenal-win-the-202526-english-premier-league"
3. DO NOT modify, shorten, or invent slugs.
4. For each market you analyze, estimate the probability of the YES outcome (0.0 to 1.0).
5. Only analyze markets you have data for. Skip others.

Return a JSON object with this EXACT structure:
{{
    "evaluations": [
        {{
            "market_slug": "COPY_EXACT_SLUG_FROM_ABOVE",
            "estimated_probability": 0.XX,
            "rationale": "brief explanation based on FPL data"
        }}
    ]
}}
"""

def get_agents(groq_api_key):
    return [
        AlphaAgent("Agent_Llama_70B", "llama-3.3-70b-versatile", groq_api_key),
        AlphaAgent("Agent_Llama_8B", "llama-3.1-8b-instant", groq_api_key)
    ]
