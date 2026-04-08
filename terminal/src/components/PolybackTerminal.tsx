import { useCallback, useEffect, useState } from "react";
import {
  Activity,
  Loader2,
  RefreshCw,
  Server,
  CheckCircle2,
  XCircle,
  AlertCircle,
} from "lucide-react";
import {
  fetchPolybackClientConfig,
  polybackRelayJson,
  type PolybackClientConfig,
  type PolybackServiceTarget,
} from "@/client/polyback-api-base";

type ProbeRow = {
  id: string;
  label: string;
  target: PolybackServiceTarget;
  path: string;
};

const PROBES: ProbeRow[] = [
  { id: "exec-actuator", label: "Executor actuator", target: "executor", path: "/actuator/health" },
  { id: "exec-poly", label: "Polymarket API health", target: "executor", path: "/api/polymarket/health" },
  {
    id: "intel-actuator",
    label: "Intelligence actuator",
    target: "intelligence",
    path: "/actuator/health",
  },
  { id: "strat", label: "Strategy status", target: "strategy", path: "/api/strategy/status" },
  { id: "ingest", label: "Ingestor status", target: "ingestor", path: "/api/ingestor/status" },
  { id: "analytics", label: "Analytics status", target: "analytics", path: "/api/analytics/status" },
  { id: "infra", label: "Infrastructure status", target: "infrastructure", path: "/api/infrastructure/status" },
];

type ProbeState = "idle" | "loading" | "ok" | "err";

type ProbeResult = {
  state: ProbeState;
  summary: string;
};

const initialProbeResults = (): Record<string, ProbeResult> =>
  Object.fromEntries(PROBES.map((p) => [p.id, { state: "idle", summary: "—" }]));

export default function PolybackTerminal() {
  const [config, setConfig] = useState<(PolybackClientConfig & { success: boolean }) | null>(null);
  const [configError, setConfigError] = useState<string | null>(null);
  const [configLoading, setConfigLoading] = useState(true);
  const [probes, setProbes] = useState<Record<string, ProbeResult>>(initialProbeResults);
  const [probesLoading, setProbesLoading] = useState(false);

  const loadConfig = useCallback(async () => {
    setConfigLoading(true);
    setConfigError(null);
    try {
      const c = await fetchPolybackClientConfig();
      setConfig(c);
    } catch (e) {
      setConfig(null);
      setConfigError(e instanceof Error ? e.message : "Config load failed");
    } finally {
      setConfigLoading(false);
    }
  }, []);

  const runProbes = useCallback(async () => {
    if (!config) return;
    setProbesLoading(true);
    const loading: Record<string, ProbeResult> = {};
    for (const p of PROBES) {
      loading[p.id] = { state: "loading", summary: "…" };
    }
    setProbes(loading);

    const entries = await Promise.all(
      PROBES.map(async (p) => {
        const r = await polybackRelayJson<unknown>(p.target, p.path, { clientConfig: config });
        const jsonStr = r.data !== undefined ? JSON.stringify(r.data) : "";
        const summary =
          r.ok && jsonStr.length > 0
            ? jsonStr.slice(0, 280) + (jsonStr.length > 280 ? "…" : "")
            : r.raw.slice(0, 200) || `HTTP ${r.status}`;
        return [
          p.id,
          { state: (r.ok ? "ok" : "err") as ProbeState, summary },
        ] as const;
      })
    );

    const done: Record<string, ProbeResult> = {};
    for (const [id, pr] of entries) {
      done[id] = pr;
    }
    setProbes(done);
    setProbesLoading(false);
  }, [config]);

  useEffect(() => {
    void loadConfig();
  }, [loadConfig]);

  useEffect(() => {
    if (config) void runProbes();
  }, [config, runProbes]);

  const statusIcon = (state: ProbeState) => {
    if (state === "loading") return <Loader2 className="w-4 h-4 animate-spin text-muted-foreground" />;
    if (state === "ok") return <CheckCircle2 className="w-4 h-4 text-success" />;
    if (state === "err") return <XCircle className="w-4 h-4 text-destructive" />;
    return <AlertCircle className="w-4 h-4 text-muted-foreground" />;
  };

  return (
    <div className="min-h-full bg-background p-6 md:p-8 max-w-5xl mx-auto">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-8">
        <div className="flex items-center gap-3">
          <div className="w-12 h-12 rounded-xl bg-primary/15 border border-primary/30 flex items-center justify-center">
            <Server className="w-6 h-6 text-primary" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-foreground font-display">Polyback MM</h1>
            <p className="text-sm text-muted-foreground">
              YAML-backed client config and probes. Default: Bun relay (
              <code className="text-xs bg-muted px-1 rounded">POLYBACK_BOOTSTRAP_URL</code>
              ). Optional browser → Go:{" "}
              <code className="text-xs bg-muted px-1 rounded">POLYBACK_BROWSER_BOOTSTRAP_URL</code>{" "}
              + polyback-mm <code className="text-xs bg-muted px-1 rounded">cors_allowed_origins</code>.
            </p>
          </div>
        </div>
        <button
          type="button"
          onClick={() => {
            void loadConfig().then(() => void runProbes());
          }}
          disabled={configLoading || probesLoading}
          className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-card terminal-border border-border hover:border-primary/50 transition-colors text-sm font-medium disabled:opacity-50"
        >
          <RefreshCw className={`w-4 h-4 ${configLoading || probesLoading ? "animate-spin" : ""}`} />
          Refresh all
        </button>
      </div>

      {/* Client config */}
      <div className="rounded-xl terminal-border bg-card p-6 mb-6">
        <div className="flex items-center gap-2 mb-4">
          <Activity className="w-4 h-4 text-primary" />
          <h2 className="text-sm font-mono uppercase tracking-wider text-primary">Client config</h2>
        </div>
        {configLoading && (
          <div className="flex items-center gap-2 text-muted-foreground text-sm">
            <Loader2 className="w-4 h-4 animate-spin" />
            Loading from Go…
          </div>
        )}
        {configError && (
          <div className="rounded-lg bg-destructive/10 border border-destructive/30 p-4 text-sm text-destructive">
            {configError}
            <p className="mt-2 text-muted-foreground text-xs">
              Start executor (or any polyback HTTP process) with{" "}
              <code className="bg-muted px-1 rounded">POLYBACK_CONFIG</code> pointing at your YAML. Bootstrap
              defaults to <code className="bg-muted px-1 rounded">http://127.0.0.1:8080</code>.
            </p>
          </div>
        )}
        {config && !configLoading && (
          <div className="space-y-4 text-sm">
            <div className="grid sm:grid-cols-2 gap-3">
              <div>
                <div className="text-xs text-muted-foreground uppercase font-mono mb-1">apiBaseUrl</div>
                <div className="font-mono text-foreground break-all">{config.apiBaseUrl}</div>
              </div>
              <div>
                <div className="text-xs text-muted-foreground uppercase font-mono mb-1">hftMode</div>
                <div className="font-mono text-foreground">{config.hftMode ?? "—"}</div>
              </div>
            </div>
            {config.serviceUrls && (
              <div>
                <div className="text-xs text-muted-foreground uppercase font-mono mb-2">serviceUrls</div>
                <div className="grid gap-2 font-mono text-xs">
                  {(Object.entries(config.serviceUrls) as [string, string][]).map(([k, v]) =>
                    v ? (
                      <div key={k} className="flex flex-wrap gap-2 border border-border/50 rounded-md px-3 py-2 bg-secondary/20">
                        <span className="text-primary">{k}</span>
                        <span className="text-foreground break-all">{v}</span>
                      </div>
                    ) : null
                  )}
                </div>
              </div>
            )}
            {config.modules && config.modules.length > 0 && (
              <div>
                <div className="text-xs text-muted-foreground uppercase font-mono mb-2">modules</div>
                <ul className="space-y-1 text-xs font-mono">
                  {config.modules.map((m) => (
                    <li key={m.name} className="text-foreground">
                      <span className="text-primary">{m.name}</span>{" "}
                      <span className="text-muted-foreground">{m.pathPrefix}</span>
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Health probes */}
      <div className="rounded-xl terminal-border bg-card p-6">
        <div className="flex items-center gap-2 mb-4">
          <Activity className="w-4 h-4 text-success" />
          <h2 className="text-sm font-mono uppercase tracking-wider text-success">Service probes</h2>
        </div>
        {!config && !configLoading && (
          <p className="text-sm text-muted-foreground">Load client config first.</p>
        )}
        {config && (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-xs text-muted-foreground uppercase font-mono border-b border-border">
                  <th className="pb-2 pr-4">Status</th>
                  <th className="pb-2 pr-4">Check</th>
                  <th className="pb-2 pr-4">Target</th>
                  <th className="pb-2">Path</th>
                  <th className="pb-2">Response</th>
                </tr>
              </thead>
              <tbody>
                {PROBES.map((p) => {
                  const row = probes[p.id] ?? { state: "idle", summary: "—" };
                  return (
                    <tr key={p.id} className="border-b border-border/50 align-top">
                      <td className="py-3 pr-4">{statusIcon(row.state)}</td>
                      <td className="py-3 pr-4 text-foreground">{p.label}</td>
                      <td className="py-3 pr-4 font-mono text-xs text-primary">{p.target}</td>
                      <td className="py-3 pr-4 font-mono text-xs text-muted-foreground">{p.path}</td>
                      <td className="py-3 font-mono text-xs text-foreground/90 break-all max-w-md">
                        {row.summary}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
