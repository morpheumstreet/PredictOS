Here's a clear, practical outline of the **actual APIs you need** to gather **all the rules and context** (market questions, descriptions, resolution criteria, edge cases, end dates, outcomes, categories, tags, events, etc.) from Polymarket.

### 1. Primary API: **Gamma API** (This is what you need most)
- **Base URL**: `https://gamma-api.polymarket.com`
- **Purpose**: This is the main public API for **market discovery, metadata, rules & context**.
  - It provides rich details that appear on the website (question, long description, resolution logic, outcomes, end dates, tags, categories, images, etc.).
  - Events group related markets and often include higher-level context.
- **Does it require an API key?**  
  **No.** It is **fully public** — no authentication, no API key, no wallet needed. You can call it directly with simple `GET` requests (curl, Python requests, etc.).

**Key endpoints you should use** (all GET requests):

| Endpoint | What it gives you | Recommended parameters | Why it's useful for rules/context |
|----------|-------------------|------------------------|-----------------------------------|
| `/markets` | List of markets (paginated) | `active=true`, `closed=false`, `limit=100`, `offset=0`, `slug=...`, `tag_id=...` | Bulk fetch markets with descriptions, outcomes, end dates, etc. |
| `/markets/{id}` or `/markets?slug=market-slug` | Full details for one specific market | Use ID or slug from the list | Deepest context: full `description` field usually contains resolution rules and edge cases |
| `/events` | List of events (groups of markets) | `active=true`, `closed=false`, `limit=100` | Higher-level context; events often have broader rules/descriptions |
| `/events/{id}` or `/events?slug=event-slug` | Full details for one event | Use ID or slug | Nested markets + event-level rules |
| `/public-search` | Search across markets & events | `query=your search term` | Quick discovery |
| `/tags` | List of tags/categories | — | Filter by topic (e.g., politics, sports, crypto) |
| `/series` | Recurring event series | — | Context for ongoing topics |

**Best strategy**:
- Start with `/events?active=true&closed=false&limit=100` (or `/markets`) and paginate using `offset` until `has_more` is false.
- For any interesting market/event, fetch the detailed version by ID or slug — the `description` field is where most **rules and resolution context** lives.

### 2. Secondary API: **Data API** (Optional, for extra context)
- **Base URL**: `https://data-api.polymarket.com`
- **Purpose**: User positions, trades, activity, open interest, top holders, leaderboards.
- **Does it require an API key?**  
  **No** for most public reads (e.g., open interest, trades for a market). It is also fully public.

Useful if you want trading volume, liquidity, or historical activity as supporting context for a market’s rules.

### 3. CLOB API (Only if you need prices/order books)
- **Base URL**: `https://clob.polymarket.com`
- **Purpose**: Live prices, order books, spreads, price history.
- **Does it require an API key?**  
  **No** for public read endpoints (e.g., `/market-data/get-order-book`, midpoint prices, spreads).  
  **Yes** only for trading actions (placing/canceling orders).

You probably don’t need this for “rules and context”.

### Summary: What you actually need
- **Main tool**: **Gamma API** (`https://gamma-api.polymarket.com`) — covers **95%+** of rules, descriptions, resolution criteria, events, and market metadata.
- **No API key required** for any of the read-only data endpoints above.
- **No spider/crawler needed** for market rules & context — the API gives you clean, structured JSON directly.

### Quick examples (copy-paste)
```bash
# All active markets (start here)
curl "https://gamma-api.polymarket.com/markets?active=true&closed=false&limit=100"

# Full details for one market by slug
curl "https://gamma-api.polymarket.com/markets?slug=will-trump-win-2028-presidential-election"

# All active events
curl "https://gamma-api.polymarket.com/events?active=true&closed=false&limit=100"
```

Would you like me to provide a **complete ready-to-run Python script** that:
- Fetches all markets/events with pagination?
- Saves everything (including full descriptions/rules) to JSONL or JSON files?
- Optionally filters by tags or keywords?

Just say yes and tell me your preferred output format (e.g., one big file, per-market files, etc.), and I’ll give you the exact code.