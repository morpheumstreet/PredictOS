
import { useState } from "react";
import { Loader2, ScanSearch, AlertTriangle } from "lucide-react";
import type { GetEventsResponse } from "@/types/agentic";

function marketTitle(m: unknown): string {
  if (m && typeof m === "object") {
    const o = m as Record<string, unknown>;
    for (const key of ["title", "question", "name"] as const) {
      const v = o[key];
      if (typeof v === "string" && v.trim()) return v;
    }
  }
  return "Untitled market";
}

export default function EventScannerTerminal() {
  const [url, setUrl] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [data, setData] = useState<GetEventsResponse | null>(null);

  const scan = async () => {
    const trimmed = url.trim();
    if (!trimmed) {
      setError("Enter a Polymarket or Kalshi event URL");
      return;
    }
    setLoading(true);
    setError(null);
    setData(null);
    try {
      const res = await fetch("/api/get-events", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ url: trimmed }),
      });
      const json: GetEventsResponse = await res.json();
      if (!json.success) {
        setError(json.error || "Failed to resolve event URL");
        return;
      }
      setData(json);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Request failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-[calc(100vh-80px)] px-2 py-4 md:px-4 md:py-6">
      <div className="max-w-4xl mx-auto space-y-6">
        <div className="text-center py-8 fade-in">
          <h2 className="font-display text-xl md:text-2xl font-bold text-primary text-glow mb-1">
            Event Scanner
          </h2>
          <p className="text-muted-foreground max-w-lg mx-auto">
            Resolve a prediction-market URL into normalized event metadata and markets (same pipeline as Super Intelligence).
          </p>
        </div>

        <div className="relative z-20 border border-border rounded-lg bg-card/80 backdrop-blur-sm border-glow">
          <div className="flex items-center gap-2 px-4 py-2 border-b border-border/50">
            <ScanSearch className="w-4 h-4 text-primary" />
            <span className="text-xs text-muted-foreground font-display">GET EVENTS</span>
          </div>
          <div className="p-4 space-y-4">
            <div className="flex flex-col sm:flex-row gap-3 sm:items-center">
              <label className="text-sm font-medium text-muted-foreground shrink-0 sm:min-w-[100px]">
                Event URL
              </label>
              <input
                type="url"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                placeholder="https://polymarket.com/... or Kalshi event link"
                className="flex-1 w-full px-4 py-3 rounded-lg bg-secondary/50 border border-border text-sm font-mono hover:border-primary/50 transition-all placeholder:text-muted-foreground/50 focus:outline-none focus:border-primary"
              />
              <button
                type="button"
                onClick={scan}
                disabled={loading}
                className="shrink-0 px-5 py-3 rounded-lg bg-primary text-primary-foreground font-display text-sm font-semibold hover:opacity-90 transition-opacity disabled:opacity-50 flex items-center justify-center gap-2"
              >
                {loading ? (
                  <>
                    <Loader2 className="w-4 h-4 animate-spin" />
                    Scanning
                  </>
                ) : (
                  <>
                    <ScanSearch className="w-4 h-4" />
                    Scan
                  </>
                )}
              </button>
            </div>

            {error && (
              <div className="flex items-start gap-2 rounded-lg border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5" />
                <span>{error}</span>
              </div>
            )}
          </div>
        </div>

        {data && (
          <div className="border border-border rounded-lg bg-card/80 backdrop-blur-sm terminal-border space-y-4 p-4">
            <div className="grid gap-2 text-sm font-mono sm:grid-cols-2">
              <div>
                <span className="text-muted-foreground">Platform</span>
                <p className="text-foreground">{data.pmType ?? "—"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">Markets</span>
                <p className="text-foreground">{data.markets?.length ?? data.marketsCount ?? 0}</p>
              </div>
              {data.eventIdentifier && (
                <div className="sm:col-span-2">
                  <span className="text-muted-foreground">Event identifier</span>
                  <p className="text-foreground break-all">{data.eventIdentifier}</p>
                </div>
              )}
              {data.eventId && (
                <div className="sm:col-span-2">
                  <span className="text-muted-foreground">Event ID</span>
                  <p className="text-foreground break-all">{data.eventId}</p>
                </div>
              )}
              {data.urlSource && (
                <div>
                  <span className="text-muted-foreground">URL source</span>
                  <p className="text-foreground">{data.urlSource}</p>
                </div>
              )}
            </div>

            <div>
              <h3 className="text-xs font-display text-muted-foreground uppercase tracking-wider mb-2">
                Markets
              </h3>
              {!data.markets?.length ? (
                <p className="text-sm text-muted-foreground">No markets in response.</p>
              ) : (
                <ul className="space-y-2 max-h-[min(50vh,28rem)] overflow-y-auto pr-1">
                  {data.markets.map((m, i) => (
                    <li
                      key={i}
                      className="rounded-md border border-border/60 bg-secondary/20 px-3 py-2 text-sm text-foreground"
                    >
                      {marketTitle(m)}
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
