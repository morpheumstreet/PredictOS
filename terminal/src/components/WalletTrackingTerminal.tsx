import { useState, useEffect, useCallback } from "react";
import { Play, Square, Eye, AlertTriangle, Plus } from "lucide-react";
import WalletTrackerBotBar from "@/components/WalletTrackerBotBar";
import WalletTrackingLogPanel from "@/components/WalletTrackingLogPanel";
import { useWalletTrackerBots } from "@/hooks/useWalletTrackerBots";
import type { WalletTrackerBackendStatus } from "@/types/wallet-tracking";

type WalletTrackingStatusResponse = {
  ok: boolean;
  dome_configured: boolean;
  active_stream_connections?: number;
  dome_key_count?: number;
  max_concurrent_streams?: number;
  max_subscriptions_per_key?: number;
  slots_in_use?: number;
};

const WalletTrackingTerminal = () => {
  const [backendStatus, setBackendStatus] = useState<WalletTrackerBackendStatus>("checking");
  const [capacityHint, setCapacityHint] = useState<string | null>(null);
  const backendReady = backendStatus === "ready";

  const {
    botsList,
    selectedKey,
    setSelectedKey,
    selectedBot,
    addBot,
    removeBot,
    startBot,
    stopBot,
    clearLogsForSelected,
    anyRunning,
  } = useWalletTrackerBots(backendReady);

  const [newWalletInput, setNewWalletInput] = useState("");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const res = await fetch("/api/wallet-tracking/status");
        const data = (await res.json()) as Partial<WalletTrackingStatusResponse>;
        if (cancelled) return;
        if (data.dome_configured) {
          setBackendStatus("ready");
          const nKeys = data.dome_key_count ?? 1;
          const maxStreams = data.max_concurrent_streams ?? nKeys * (data.max_subscriptions_per_key ?? 2);
          const inUse = data.slots_in_use ?? data.active_stream_connections ?? 0;
          setCapacityHint(`${nKeys} Dome key(s) · ${inUse}/${maxStreams} stream slots in use (auto-balanced across keys)`);
        } else {
          setBackendStatus("misconfigured");
          setCapacityHint(null);
        }
      } catch {
        if (!cancelled) {
          setBackendStatus("error");
          setCapacityHint(null);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const handleAddBot = useCallback(() => {
    setError(null);
    const result = addBot(newWalletInput);
    if (!result.ok) {
      setError(result.error ?? "Could not add wallet.");
      return;
    }
    setNewWalletInput("");
  }, [addBot, newWalletInput]);

  const handleStartSelected = useCallback(() => {
    if (!selectedKey) {
      setError("Select a tracker first.");
      return;
    }
    setError(null);
    const result = startBot(selectedKey);
    if (!result.ok) {
      setError(result.error ?? "Could not start tracking.");
    }
  }, [selectedKey, startBot]);

  const handleStopSelected = useCallback(() => {
    if (!selectedKey) return;
    setError(null);
    stopBot(selectedKey);
  }, [selectedKey, stopBot]);

  return (
    <div className="min-h-[calc(100vh-80px)] px-2 py-4 md:px-4 md:py-6">
      <div className="max-w-4xl mx-auto">
        <div className="space-y-6">
          <div className="text-center py-8 fade-in">
            <div className="relative mb-8">
              <h2 className="font-display text-xl md:text-2xl font-bold text-primary text-glow mb-1">
                Polymarket Wallet Tracking
              </h2>
              <p className="text-muted-foreground max-w-lg mx-auto">
                Track real-time order activity on any Polymarket wallet using Dome API WebSocket. Add multiple
                wallets; each has its own stream and activity log.
              </p>
            </div>
          </div>

          <div className="relative z-20 border border-border rounded-lg bg-card/80 backdrop-blur-sm border-glow">
            <div className="flex items-center justify-between px-4 py-2 border-b border-border/50 gap-3 flex-wrap">
              <div className="flex items-center gap-2">
                <Eye className="w-4 h-4 text-primary" />
                <span className="text-xs text-muted-foreground font-display">WALLET TRACKER</span>
              </div>
              <div className="flex items-center gap-3 ml-auto">
                {backendStatus === "checking" && (
                  <span className="text-xs text-muted-foreground font-mono">Backend…</span>
                )}
                {backendStatus === "ready" && !anyRunning && (
                  <div className="flex flex-col items-end gap-0.5 min-w-0">
                    <span className="text-xs text-green-500/90 font-mono">Backend ready</span>
                    {capacityHint && (
                      <span className="text-[10px] text-muted-foreground font-mono text-right max-w-[min(100%,18rem)] leading-tight">
                        {capacityHint}
                      </span>
                    )}
                  </div>
                )}
                {backendStatus === "ready" && anyRunning && (
                  <div className="flex flex-col items-end gap-0.5 min-w-0">
                    <div className="flex items-center gap-2">
                      <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
                      <span className="text-xs text-green-500 font-mono">TRACKING</span>
                    </div>
                    {capacityHint && (
                      <span className="text-[10px] text-muted-foreground font-mono text-right max-w-[min(100%,18rem)] leading-tight">
                        {capacityHint}
                      </span>
                    )}
                  </div>
                )}
                {backendStatus === "misconfigured" && (
                  <span className="text-xs text-warning font-mono">DOME_API_KEY missing</span>
                )}
                {backendStatus === "error" && (
                  <span className="text-xs text-destructive font-mono">Status unreachable</span>
                )}
              </div>
            </div>

            <WalletTrackerBotBar
              bots={botsList}
              selectedKey={selectedKey}
              backendReady={backendReady}
              onSelect={setSelectedKey}
              onStart={(key) => {
                setError(null);
                const r = startBot(key);
                if (!r.ok) setError(r.error ?? "Could not start.");
              }}
              onStop={stopBot}
              onRemove={removeBot}
            />

            <div className="p-4 space-y-4">
              <div className="flex flex-col sm:flex-row items-start sm:items-end gap-3">
                <div className="flex-1 w-full space-y-1.5">
                  <label className="text-sm font-medium text-muted-foreground">Add wallet</label>
                  <input
                    type="text"
                    value={newWalletInput}
                    onChange={(e) => setNewWalletInput(e.target.value.trim())}
                    placeholder="0x… (unique; duplicates rejected)"
                    className="w-full px-4 py-3 rounded-lg bg-secondary/50 border border-border text-sm font-mono hover:border-primary/50 transition-all placeholder:text-muted-foreground/50 focus:outline-none focus:border-primary"
                  />
                </div>
                <button
                  type="button"
                  onClick={handleAddBot}
                  className="flex items-center gap-2 px-5 py-3 rounded-lg font-medium transition-all bg-secondary/80 border border-border text-foreground hover:border-primary/50 shrink-0"
                >
                  <Plus className="w-4 h-4" />
                  <span>Add</span>
                </button>
              </div>

              {selectedKey ? (
                <div className="rounded-lg border border-border/60 bg-secondary/20 px-4 py-3 space-y-3">
                  <div>
                    <p className="text-xs text-muted-foreground mb-1">Selected wallet</p>
                    <p className="text-sm font-mono break-all text-foreground">{selectedKey}</p>
                  </div>
                  <div className="flex flex-wrap gap-3">
                    {!selectedBot?.isRunning ? (
                      <button
                        type="button"
                        onClick={handleStartSelected}
                        disabled={!backendReady}
                        className="flex items-center gap-2 px-6 py-3 rounded-lg font-medium transition-all bg-primary/20 border border-primary/50 text-primary hover:bg-primary/30 glow-box-hover disabled:opacity-50 disabled:pointer-events-none"
                      >
                        <Play className="w-4 h-4" />
                        <span>Start tracking</span>
                      </button>
                    ) : (
                      <button
                        type="button"
                        onClick={handleStopSelected}
                        className="flex items-center gap-2 px-6 py-3 rounded-lg font-medium transition-all bg-destructive/20 border border-destructive/50 text-destructive hover:bg-destructive/30"
                      >
                        <Square className="w-4 h-4" />
                        <span>Stop tracking</span>
                      </button>
                    )}
                  </div>
                </div>
              ) : null}
            </div>
          </div>

          {error && (
            <div className="border border-destructive/50 rounded-lg bg-destructive/10 p-4 fade-in">
              <div className="flex items-center gap-2">
                <AlertTriangle className="w-4 h-4 text-destructive" />
                <p className="text-destructive text-sm font-mono">{error}</p>
              </div>
            </div>
          )}

          <WalletTrackingLogPanel
            logs={selectedBot?.logs ?? []}
            backendStatus={backendStatus}
            selectedAddressKey={selectedKey}
            isSelectedRunning={selectedBot?.isRunning ?? false}
            onClear={clearLogsForSelected}
          />
        </div>
      </div>
    </div>
  );
};

export default WalletTrackingTerminal;
