import { useState, useEffect, useMemo } from "react";
import { Loader2, AlertTriangle, Database, RefreshCw, Clock } from "lucide-react";
import type {
  AlphaRulesSummary,
  AlphaRulesEventRow,
  AlphaRulesEventsAllResponse,
} from "@/types/alpha-rules";

function fmtNum(n: number | null | undefined): string {
  if (n == null || Number.isNaN(n)) return "—";
  if (n >= 1e6) return `${(n / 1e6).toFixed(2)}M`;
  if (n >= 1e3) return `${(n / 1e3).toFixed(1)}k`;
  return n.toFixed(0);
}

/** Next collector boundary: UTC wall slots when period divides 60 (cron-style); otherwise epoch-aligned steps. */
function nextScheduledScanUtc(from: Date, periodMinutes: number): Date {
  const period = Math.max(1, Math.min(1440, Math.floor(periodMinutes)));
  const fromMs = from.getTime();
  const stepMs = period * 60_000;
  if (period <= 60 && 60 % period === 0) {
    const dayStart = Date.UTC(
      from.getUTCFullYear(),
      from.getUTCMonth(),
      from.getUTCDate(),
      0,
      0,
      0,
      0
    );
    const msInDay = fromMs - dayStart;
    let candidate = dayStart + Math.floor(msInDay / stepMs) * stepMs + stepMs;
    if (candidate <= fromMs) candidate += stepMs;
    if (candidate >= dayStart + 86_400_000) {
      return new Date(dayStart + 86_400_000);
    }
    return new Date(candidate);
  }
  let t = Math.floor(fromMs / stepMs) * stepMs;
  if (t <= fromMs) t += stepMs;
  return new Date(t);
}

function fmtCountdown(ms: number): string {
  if (ms <= 0) return "0s";
  const s = Math.ceil(ms / 1000);
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  if (h > 0) return `${h}h ${m}m ${sec}s`;
  if (m > 0) return `${m}m ${sec}s`;
  return `${sec}s`;
}

const utcTimeFmt = new Intl.DateTimeFormat("en-GB", {
  timeZone: "UTC",
  hour: "2-digit",
  minute: "2-digit",
  second: "2-digit",
  hour12: false,
});

export default function EventScannerTerminal() {
  const [summary, setSummary] = useState<AlphaRulesSummary | null>(null);
  const [events, setEvents] = useState<AlphaRulesEventRow[]>([]);
  const [eventsMeta, setEventsMeta] = useState<{ total: number; truncated: boolean } | null>(null);
  const [dbLoading, setDbLoading] = useState(true);
  const [dbError, setDbError] = useState<string | null>(null);
  const [filter, setFilter] = useState("");
  const [nowTick, setNowTick] = useState(() => Date.now());

  useEffect(() => {
    const id = window.setInterval(() => setNowTick(Date.now()), 1000);
    return () => window.clearInterval(id);
  }, []);

  const loadDb = async () => {
    setDbLoading(true);
    setDbError(null);
    try {
      const [sumRes, evRes] = await Promise.all([
        fetch("/api/alpha-rules"),
        fetch("/api/alpha-rules?table=events&all=1"),
      ]);

      const sumJson = (await sumRes.json()) as AlphaRulesSummary;
      if (!sumRes.ok || !sumJson.success) {
        setSummary(null);
        setEvents([]);
        setEventsMeta(null);
        setDbError(
          sumJson.error ||
            sumJson.hint ||
            `Database unavailable (${sumRes.status})`
        );
        return;
      }
      setSummary(sumJson);

      const evJson = (await evRes.json()) as AlphaRulesEventsAllResponse;
      if (!evRes.ok || !evJson.success || !Array.isArray(evJson.rows)) {
        setEvents([]);
        setEventsMeta(null);
        setDbError(evJson.error || "Failed to load events");
        return;
      }
      setEvents(evJson.rows as AlphaRulesEventRow[]);
      setEventsMeta({ total: evJson.total, truncated: evJson.truncated });
    } catch (e) {
      setDbError(e instanceof Error ? e.message : "Request failed");
      setEvents([]);
      setEventsMeta(null);
    } finally {
      setDbLoading(false);
    }
  };

  useEffect(() => {
    void loadDb();
  }, []);

  const scanIntervalMinutes = summary?.scanIntervalMinutes ?? 30;
  const nextScanAt = useMemo(
    () => nextScheduledScanUtc(new Date(nowTick), scanIntervalMinutes),
    [nowTick, scanIntervalMinutes]
  );
  const msUntilNext = nextScanAt.getTime() - nowTick;

  const filteredEvents = useMemo(() => {
    const q = filter.trim().toLowerCase();
    if (!q) return events;
    return events.filter((e) => {
      const blob = [
        e.id,
        e.slug,
        e.ticker,
        e.title,
        e.description,
        e.external_truth_source_urls,
      ]
        .filter(Boolean)
        .join(" ")
        .toLowerCase();
      return blob.includes(q);
    });
  }, [events, filter]);

  return (
    <div className="min-h-[calc(100vh-80px)] px-2 py-4 md:px-4 md:py-6">
      <div className="max-w-6xl mx-auto space-y-6">
        <div className="text-center py-6 fade-in">
          <h2 className="font-display text-xl md:text-2xl font-bold text-primary text-glow mb-1">
            Event Scanner
          </h2>
          <p className="text-muted-foreground max-w-2xl mx-auto text-sm">
            All events from the alpha-rules SQLite database (
            <code className="text-xs text-primary/80">strat/alpha-rules/data/alpha_rules.sqlite</code>
            ), refreshed by the collector. Use the filter to search title, slug, or id.
          </p>
        </div>

        <div className="relative z-20 border border-border rounded-lg bg-card/80 backdrop-blur-sm border-glow">
          <div className="flex flex-wrap items-center justify-between gap-2 px-4 py-2 border-b border-border/50">
            <div className="flex items-center gap-2">
              <Database className="w-4 h-4 text-primary" />
              <span className="text-xs text-muted-foreground font-display">ALPHA RULES DB</span>
            </div>
            {!dbLoading && !dbError && summary?.counts && (
              <div
                className="flex items-center gap-2 text-xs font-mono text-muted-foreground order-last sm:order-none sm:flex-1 sm:justify-center"
                title={`Assumes collector every ${scanIntervalMinutes} min (set ALPHA_RULES_SCAN_INTERVAL_MINUTES). Next slot UTC.`}
              >
                <Clock className="w-3.5 h-3.5 text-primary shrink-0" />
                <span>
                  <span className="text-foreground/90">Next scan</span>{" "}
                  <span className="text-primary tabular-nums">{utcTimeFmt.format(nextScanAt)} UTC</span>
                  <span className="text-muted-foreground"> · </span>
                  <span className="tabular-nums">{fmtCountdown(msUntilNext)}</span>
                </span>
              </div>
            )}
            <button
              type="button"
              onClick={() => void loadDb()}
              disabled={dbLoading}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-mono border border-border hover:border-primary/50 hover:bg-secondary/50 transition-colors disabled:opacity-50"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${dbLoading ? "animate-spin" : ""}`} />
              Reload
            </button>
          </div>
          <div className="p-4 space-y-4">
            {dbLoading && (
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <Loader2 className="w-4 h-4 animate-spin text-primary" />
                Loading events from database…
              </div>
            )}

            {dbError && !dbLoading && (
              <div className="flex items-start gap-2 rounded-lg border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5" />
                <div>
                  <p>{dbError}</p>
                  {summary?.path == null && (
                    <p className="text-xs mt-1 opacity-80">
                      Set <code className="text-destructive">ALPHA_RULES_DB</code> in{" "}
                      <code className="text-destructive">.env</code> if the file lives elsewhere.
                    </p>
                  )}
                </div>
              </div>
            )}

            {!dbLoading && !dbError && summary?.counts && (
              <div className="flex flex-wrap gap-3 text-xs font-mono text-muted-foreground">
                <span>
                  <span className="text-foreground font-semibold">{eventsMeta?.total ?? events.length}</span>{" "}
                  events
                  {eventsMeta?.truncated ? (
                    <span className="text-warning ml-1">(showing first 50k)</span>
                  ) : null}
                </span>
                <span>·</span>
                <span>{summary.counts.markets ?? 0} markets</span>
                <span>·</span>
                <span>{summary.counts.market_outcomes ?? 0} outcomes</span>
                {summary.lastScanRun && (
                  <>
                    <span>·</span>
                    <span>
                      Last scan: {summary.lastScanRun.status}
                      {summary.lastScanRun.finished_at
                        ? ` @ ${summary.lastScanRun.finished_at}`
                        : ""}
                    </span>
                  </>
                )}
              </div>
            )}

            {!dbLoading && !dbError && (
              <div>
                <label className="text-xs font-mono text-muted-foreground uppercase tracking-wider block mb-2">
                  Filter events
                </label>
                <input
                  type="search"
                  value={filter}
                  onChange={(e) => setFilter(e.target.value)}
                  placeholder="Search title, slug, ticker, id…"
                  className="w-full px-4 py-2.5 rounded-lg bg-secondary/50 border border-border text-sm font-mono hover:border-primary/50 transition-all placeholder:text-muted-foreground/50 focus:outline-none focus:border-primary"
                />
                <p className="text-[11px] text-muted-foreground mt-1.5">
                  Showing {filteredEvents.length} of {events.length} loaded
                </p>
              </div>
            )}
          </div>
        </div>

        {!dbLoading && !dbError && events.length > 0 && (
          <div className="border border-border rounded-lg bg-card/80 backdrop-blur-sm terminal-border overflow-hidden">
            <div className="max-h-[min(70vh,42rem)] overflow-auto">
              <table className="w-full text-left text-sm border-collapse">
                <thead className="sticky top-0 z-[1] bg-card/95 backdrop-blur-sm border-b border-border">
                  <tr className="font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
                    <th className="px-3 py-2 font-medium w-[36%]">Title</th>
                    <th className="px-3 py-2 font-medium">Slug</th>
                    <th className="px-3 py-2 font-medium whitespace-nowrap">Profit α</th>
                    <th className="px-3 py-2 font-medium whitespace-nowrap">Volume</th>
                    <th className="px-3 py-2 font-medium whitespace-nowrap">Status</th>
                    <th className="px-3 py-2 font-medium whitespace-nowrap">Last scan</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredEvents.map((e) => (
                    <tr
                      key={e.id}
                      className="border-b border-border/40 hover:bg-secondary/20 transition-colors"
                    >
                      <td className="px-3 py-2 align-top">
                        <p className="text-foreground font-medium line-clamp-2">
                          {e.title?.trim() || "—"}
                        </p>
                        <p className="text-[10px] text-muted-foreground font-mono truncate max-w-[28rem] mt-0.5">
                          {e.id}
                        </p>
                      </td>
                      <td className="px-3 py-2 align-top text-xs font-mono text-muted-foreground break-all max-w-[10rem]">
                        {e.slug || "—"}
                      </td>
                      <td className="px-3 py-2 align-top">
                        {e.has_profit_opportunity ? (
                          <span className="text-[10px] px-1.5 py-0.5 rounded bg-primary/20 text-primary font-mono">
                            yes
                          </span>
                        ) : (
                          <span className="text-[10px] text-muted-foreground">—</span>
                        )}
                      </td>
                      <td className="px-3 py-2 align-top font-mono text-xs text-muted-foreground whitespace-nowrap">
                        {fmtNum(e.volume)}
                      </td>
                      <td className="px-3 py-2 align-top whitespace-nowrap">
                        {e.closed ? (
                          <span className="text-[10px] text-muted-foreground">closed</span>
                        ) : e.active ? (
                          <span className="text-[10px] text-success">active</span>
                        ) : (
                          <span className="text-[10px] text-muted-foreground">inactive</span>
                        )}
                      </td>
                      <td className="px-3 py-2 align-top text-[11px] font-mono text-muted-foreground whitespace-nowrap">
                        {e.last_scanned_at || e.fetched_at || "—"}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {!dbLoading && !dbError && events.length === 0 && (
          <p className="text-center text-sm text-muted-foreground">No events in the database yet.</p>
        )}
      </div>
    </div>
  );
}
