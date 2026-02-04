import requests
import pandas as pd
import time
import math
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry

class DataEngine:
    def __init__(self, dome_api_key):
        self.dome_api_key = dome_api_key
        self.fpl_base_url = "https://fantasy.premierleague.com/api/"
        self.base_url = "https://api.domeapi.io/v1"
        self.headers = {'User-Agent': 'Mozilla/5.0', 'Authorization': f'Bearer {self.dome_api_key}'}
        self.session = requests.Session()
        retry = Retry(total=5, backoff_factor=1, status_forcelist=[500, 502, 503, 504])
        self.session.mount('https://', HTTPAdapter(max_retries=retry))

    def get_fpl_summary(self):
        try:
            response = self.session.get(f"{self.fpl_base_url}bootstrap-static/", headers={'User-Agent': 'Mozilla/5.0'})
            data = response.json()
            teams = pd.DataFrame(data['teams'])
            players = pd.DataFrame(data['elements'])
            players['form'] = pd.to_numeric(players['form'])
            team_form = players.groupby('team')['form'].mean().to_dict()
            return {
                "teams": teams[['id', 'name', 'strength']].to_dict('records'),
                "team_form": {k: round(v, 2) for k, v in team_form.items()}
            }
        except: return {"teams": [], "team_form": {}}

    def get_upcoming_fixtures(self):
        try:
            response = self.session.get(f"{self.fpl_base_url}fixtures/", headers={'User-Agent': 'Mozilla/5.0'})
            fixtures = response.json()
            return [{"h": f['team_h'], "a": f['team_a'], "h_diff": f['team_h_difficulty']} for f in fixtures if not f['finished']][:5]
        except: return []

    def classify_market(self, title):
        title = title.lower()
        if "win the 202" in title and "premier league" in title: return "TITLE"
        if "top 4" in title or "top-4" in title or "champions league" in title: return "TOP_4"
        if "relegated" in title or "relegation" in title or "last place" in title or "bottom" in title: return "RELEGATION"
        if any(p in title for p in ["haaland", "saka", "salah", "goalscorer", "assists", "goals", "clean sheet"]): return "PLAYER_PROP"
        if "manager" in title or "sacked" in title: return "MANAGER"
        if "points" in title or "finish" in title: return "PLACEMENT"
        return "PLACEMENT"

    def get_volatility(self, condition_id):
        try:
            url = f"{self.base_url}/polymarket/candlesticks/{condition_id}?interval=1h"
            response = self.session.get(url, headers=self.headers, timeout=5)
            candles = response.json().get('candlesticks', [])
            if len(candles) < 2: return 0.05
            prices = [float(c['price']['close']) for c in candles]
            mean = sum(prices) / len(prices)
            std_dev = math.sqrt(sum((p - mean) ** 2 for p in prices) / len(prices))
            return round(std_dev, 4)
        except: return 0.05

    def search_markets_by_query(self, query):
        """Search Dome for specific market types."""
        url = f"{self.base_url}/polymarket/markets?status=open&search={query}&limit=50"
        try:
            response = self.session.get(url, headers=self.headers, timeout=10)
            return response.json().get('markets', [])
        except:
            return []

    def get_polymarket_football_markets(self):
        """
        DOME-FIRST PIPELINE with DIVERSE MARKET DISCOVERY:
        Search for multiple market types to ensure diversity.
        """
        all_markets = []
        seen_slugs = set()
        
        # Search for diverse market types
        search_queries = [
            "Premier League",
            "Premier League top 4",
            "Premier League relegated",
            "Premier League relegation",
            "Haaland goals",
            "Salah goals",
            "Premier League manager",
            "Premier League points"
        ]
        
        for query in search_queries:
            markets = self.search_markets_by_query(query)
            for m in markets:
                slug = m.get('market_slug', '') or m.get('slug', '')
                if slug and slug not in seen_slugs:
                    seen_slugs.add(slug)
                    all_markets.append(m)
        
        # EPL filter to remove non-EPL markets
        epl_keywords = ["premier league", "epl", "arsenal", "man city", "liverpool", "chelsea", "man utd", "tottenham", "leeds", "everton", "fulham", "wolves", "palace", "brentford", "brighton", "villa", "newcastle", "forest", "bournemouth", "leicester", "ipswich", "southampton", "relegated", "relegation", "haaland", "salah", "saka"]
        
        scored_markets = []
        for m in all_markets:
            title_low = m.get('title', '').lower()
            # Soft filter: only include EPL-related markets
            if not any(kw in title_low for kw in epl_keywords):
                continue
            # Exclude political markets that might have leaked in
            if any(x in title_low for x in ["nomination", "president", "election", "democratic", "republican"]):
                continue
            
            vol = self.get_volatility(m['condition_id'])
            price = m.get('price', 0.5)
            volume = m.get('volume_total', 1000)
            opp_score = vol * math.log10(max(10, volume)) * (1.1 - price)
            slug = m.get('market_slug', '') or m.get('slug', '')
            
            scored_markets.append({
                "slug": slug,
                "title": m.get('title', ''),
                "condition_id": m.get('condition_id', ''),
                "price": price,
                "volume_total": volume,
                "volatility": vol,
                "opportunity_score": round(opp_score, 3),
                "market_type": self.classify_market(m.get('title', ''))
            })
        
        # Sort by opportunity score and return top 25
        ranked = sorted(scored_markets, key=lambda x: x['opportunity_score'], reverse=True)
        return ranked[:25]

    def get_alpha_bundle(self):
        markets = self.get_polymarket_football_markets()
        return {
            "timestamp": time.time(),
            "fpl_stats": self.get_fpl_summary(),
            "fixtures": self.get_upcoming_fixtures(),
            "available_markets": markets
        }
