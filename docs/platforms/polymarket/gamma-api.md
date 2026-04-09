# Polymarket public APIs: metadata, rules, and prices

Use these APIs when you need **market questions, descriptions, resolution criteria, outcomes, dates, tags, and events** without going through a paid aggregator. This complements **Dome** (used elsewhere in PredictOS for unified feeds and execution-related data): Gamma is the canonical **public** metadata surface for Polymarket’s own stack.

## 1. Gamma API (primary for rules and context)

- **Base URL:** `https://gamma-api.polymarket.com`
- **Role:** Discovery and rich **metadata** (what you see on the site: question, long description, resolution logic, outcomes, end time, tags, categories, images). Events group related markets.
- **Authentication:** None for public reads. No API key for typical `GET` discovery and detail calls.

### Endpoints (all `GET`)

| Endpoint | Use | Typical query params | Why it matters |
|----------|-----|----------------------|----------------|
| `/markets` | Paginated market list | `active=true`, `closed=false`, `limit`, `offset`, `slug`, `tag_id` | Bulk load with descriptions and outcomes |
| `/markets/{id}` or `/markets?slug=…` | One market | id or slug | Full `description` often holds resolution rules and edge cases |
| `/events` | Event list | `active=true`, `closed=false`, `limit` | Broader context; nested markets |
| `/events/{id}` or `/events?slug=…` | One event | id or slug | Event-level copy + child markets |
| `/public-search` | Text search | `query=` | Discovery |
| `/tags` | Tags / categories | — | Filter by topic |
| `/series` | Recurring series | — | Ongoing narratives |

**Practical approach:** Start from `/events?active=true&closed=false&limit=100` or `/markets` with the same filters, paginate with `offset` until `has_more` is false. For anything you care about, refetch the **detail** by id or slug so you get the full `description`.

## 2. Data API (optional context)

- **Base URL:** `https://data-api.polymarket.com`
- **Role:** Positions, trades, activity, open interest, holders, leaderboards.
- **Authentication:** Public reads typically need no key.

Use this when volume, liquidity, or recent trade activity should inform analysis alongside rules text.

## 3. CLOB API (prices and books)

- **Base URL:** `https://clob.polymarket.com`
- **Role:** Live prices, order books, spreads, price history.
- **Authentication:** Public read endpoints generally need no key; **trading** (place/cancel) requires auth.

You usually **do not** need the CLOB for “rules and resolution text” only; you need it for executable pricing and depth.

## Summary

| Need | API |
|------|-----|
| Rules, descriptions, events, tags | **Gamma** |
| Activity / OI / trades context | **Data** |
| Books and trading | **CLOB** |

## Examples

```bash
curl "https://gamma-api.polymarket.com/markets?active=true&closed=false&limit=100"

curl "https://gamma-api.polymarket.com/markets?slug=will-trump-win-2028-presidential-election"

curl "https://gamma-api.polymarket.com/events?active=true&closed=false&limit=100"
```

In-repo client code may wrap Gamma (for example under `mm/polyback-mm/internal/platforms/polymarket/gamma/`); this page is the **contract-level** reference.

## See also

- [guides/super-intelligence.md](../../guides/super-intelligence.md) and [guides/market-analysis.md](../../guides/market-analysis.md) — Dome- and UI-oriented flows
- [operations/polyback-mm/integration.md](../../operations/polyback-mm/integration.md) — how polyback-mm consumes Polymarket data in strategies
