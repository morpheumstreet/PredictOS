# Mapping FPL Team IDs to common names used in Polymarket slugs/search
TEAM_MAPPING = {
    1: {"name": "Arsenal", "search_terms": ["Arsenal"]},
    2: {"name": "Aston Villa", "search_terms": ["Aston Villa"]},
    3: {"name": "Bournemouth", "search_terms": ["Bournemouth"]},
    4: {"name": "Brentford", "search_terms": ["Brentford"]},
    5: {"name": "Brighton", "search_terms": ["Brighton", "Hove Albion"]},
    6: {"name": "Chelsea", "search_terms": ["Chelsea"]},
    7: {"name": "Crystal Palace", "search_terms": ["Crystal Palace"]},
    8: {"name": "Everton", "search_terms": ["Everton"]},
    9: {"name": "Fulham", "search_terms": ["Fulham"]},
    10: {"name": "Ipswich", "search_terms": ["Ipswich"]},
    11: {"name": "Leicester", "search_terms": ["Leicester"]},
    12: {"name": "Liverpool", "search_terms": ["Liverpool"]},
    13: {"name": "Man City", "search_terms": ["Man City", "Manchester City"]},
    14: {"name": "Man Utd", "search_terms": ["Man Utd", "Manchester United"]},
    15: {"name": "Newcastle", "search_terms": ["Newcastle"]},
    16: {"name": "Nott'm Forest", "search_terms": ["Nottingham Forest", "Forest"]},
    17: {"name": "Southampton", "search_terms": ["Southampton"]},
    18: {"name": "Spurs", "search_terms": ["Spurs", "Tottenham"]},
    19: {"name": "West Ham", "search_terms": ["West Ham"]},
    20: {"name": "Wolves", "search_terms": ["Wolves", "Wolverhampton"]}
}

def get_team_name(team_id):
    return TEAM_MAPPING.get(team_id, {}).get("name", "Unknown")

def get_search_terms(team_id):
    return TEAM_MAPPING.get(team_id, {}).get("search_terms", [])
