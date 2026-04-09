# Market making on prediction markets: risks and defenses

Prediction-market **market making** is exposed to informed flow, fast repricing, and adversarial traders who lift stale quotes. This note captures the **failure mode** and the **defense mechanisms** that inform polyback-mm design; it is not financial advice and does not guarantee profitability after fees.

## Adversarial scenario (stale quote)

1. A market maker posts a tight two-sided book (for example YES bid 0.48 / ask 0.52).
2. A taker hits the ask at 0.52.
3. Fair value jumps (news, correlated market move, or resolution-relevant information).
4. The maker still shows 0.52 on the ask and gets picked off, or inventory becomes one-sided at a bad level.

**Takeaway:** quoting without **fast data**, **volatility-aware width**, and **inventory / toxicity controls** tends to lose to latency and information edge.

## Defense mechanisms (conceptual)

| Mechanism | Intent |
|-----------|--------|
| **Fast price discovery** | WebSocket-driven book and trade updates instead of slow polling. |
| **Volatility-adjusted spread** | Widen quotes when short-term volatility spikes (for example EWMA on returns or trade flow). |
| **Depth and liquidity monitoring** | If one side of the book thins abruptly, pause or skew that side. |
| **Event / resolution awareness** | Reduce risk into known uncertainty (resolution updates, schedule boundaries); optional automated flatten or cancel. |
| **Position and notional limits** | Cap per-side and total risk so one sequence of fills cannot blow the book. |
| **Inventory skew** | Shift reservation price / skew quotes to mean-revert inventory without naive symmetry. |
| **Order splitting** | Avoid broadcasting large intent in one clip when size would move the market against you. |
| **Toxic flow detection** | Treat clusters of aggressive flow against you as a signal to widen or stop. |
| **Anti-prediction jitter** | Small controlled noise on quotes makes pure reverse-engineering of your rule set harder (does not remove edge risk). |

## Where this lands in PredictOS

Implementation detail, package map, and config keys live in [integration.md](integration.md). Market making in polyback-mm is **off by default** in typical configs; expect parameter tuning and fee math before assuming edge.

## See also

- [integration.md](integration.md) — code map, YAML knobs, terminal relay
- [../../architecture/sqlite-vs-clickhouse.md](../../architecture/sqlite-vs-clickhouse.md) — where telemetry and analytics should live
