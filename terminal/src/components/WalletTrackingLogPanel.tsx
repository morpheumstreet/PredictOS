import { useEffect, useRef } from "react";
import type { WalletTrackingLogEntry, WalletTrackerBackendStatus } from "@/types/wallet-tracking";

type WalletTrackingLogPanelProps = {
  logs: WalletTrackingLogEntry[];
  backendStatus: WalletTrackerBackendStatus;
  selectedAddressKey: string | null;
  isSelectedRunning: boolean;
  onClear: () => void;
};

function getLogLevelStyle(level: WalletTrackingLogEntry["level"]) {
  switch (level) {
    case "SUCCESS":
      return "text-success";
    case "ERROR":
      return "text-destructive";
    case "WARN":
      return "text-warning";
    case "ORDER":
      return "text-primary";
    default:
      return "text-muted-foreground";
  }
}

function getLogLevelIcon(level: WalletTrackingLogEntry["level"]) {
  switch (level) {
    case "SUCCESS":
      return "✓";
    case "ERROR":
      return "✗";
    case "WARN":
      return "⚠";
    case "ORDER":
      return "◆";
    default:
      return "›";
  }
}

export default function WalletTrackingLogPanel({
  logs,
  backendStatus,
  selectedAddressKey,
  isSelectedRunning,
  onClear,
}: WalletTrackingLogPanelProps) {
  const logsEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    logsEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [logs]);

  const emptyMessage = !selectedAddressKey
    ? "Select a tracker above to view its activity log."
    : backendStatus === "checking"
      ? "Checking wallet tracking backend…"
      : backendStatus === "misconfigured"
        ? "Configure DOME_API_KEY to enable tracking. Logs will appear here once you start."
        : backendStatus === "error"
          ? "Could not load backend status. Logs will appear here after a successful connection."
          : "No stream events yet. Start this tracker or wait for live orders.";

  return (
    <div className="relative z-10 border border-border rounded-lg bg-card/80 backdrop-blur-sm">
      <div className="flex items-center justify-between px-4 py-2 border-b border-border/50">
        <div className="flex items-center gap-2 min-w-0">
          <div
            className={`w-2 h-2 rounded-full shrink-0 ${isSelectedRunning ? "bg-green-500" : "bg-primary"} animate-pulse`}
          />
          <span className="text-xs text-muted-foreground font-display truncate">
            ACTIVITY LOG
            {selectedAddressKey ? (
              <span className="text-muted-foreground/70 font-mono ml-2">
                {selectedAddressKey.slice(0, 6)}…{selectedAddressKey.slice(-4)}
              </span>
            ) : null}
          </span>
        </div>
        <button
          type="button"
          disabled={!selectedAddressKey}
          onClick={onClear}
          className="text-xs text-muted-foreground hover:text-foreground transition-colors disabled:opacity-40 disabled:pointer-events-none"
        >
          Clear
        </button>
      </div>

      <div className="h-[400px] overflow-y-auto p-4 font-mono text-sm">
        {logs.length === 0 ? (
          <div className="flex items-center justify-center h-full text-muted-foreground text-center px-4">
            <span>{emptyMessage}</span>
          </div>
        ) : (
          <div className="space-y-1">
            {logs.map((log, index) => (
              <div key={`${log.timestamp}-${index}`} className="flex gap-2 leading-relaxed">
                <span className="text-muted-foreground/60 text-xs whitespace-nowrap">
                  {new Date(log.timestamp).toLocaleTimeString()}
                </span>
                <span className={`${getLogLevelStyle(log.level)} w-4`}>{getLogLevelIcon(log.level)}</span>
                <span className={getLogLevelStyle(log.level)}>{log.message}</span>
              </div>
            ))}
            <div ref={logsEndRef} />
          </div>
        )}
      </div>
    </div>
  );
}
