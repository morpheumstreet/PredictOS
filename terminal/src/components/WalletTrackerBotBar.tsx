import { Play, Square, X } from "lucide-react";
import { shortWalletLabel, type WalletTrackerBot } from "@/types/wallet-tracking";

type WalletTrackerBotBarProps = {
  bots: WalletTrackerBot[];
  selectedKey: string | null;
  backendReady: boolean;
  onSelect: (addressKey: string) => void;
  onStart: (addressKey: string) => void;
  onStop: (addressKey: string) => void;
  onRemove: (addressKey: string) => void;
};

export default function WalletTrackerBotBar({
  bots,
  selectedKey,
  backendReady,
  onSelect,
  onStart,
  onStop,
  onRemove,
}: WalletTrackerBotBarProps) {
  if (bots.length === 0) {
    return (
      <div className="px-4 py-3 border-b border-border/50 text-xs text-muted-foreground font-mono">
        No trackers yet. Add a wallet address below.
      </div>
    );
  }

  return (
    <div className="border-b border-border/50 px-2 py-2">
      <div className="flex gap-2 overflow-x-auto pb-1">
        {bots.map((bot) => {
          const selected = bot.addressKey === selectedKey;
          return (
            <div
              key={bot.addressKey}
              className={`flex shrink-0 items-center gap-1.5 rounded-lg border px-2 py-1.5 transition-colors ${
                selected
                  ? "border-primary/60 bg-primary/10"
                  : "border-border/60 bg-secondary/30 hover:border-border"
              }`}
            >
              <button
                type="button"
                onClick={() => onSelect(bot.addressKey)}
                className="flex items-center gap-2 min-w-0 text-left"
                title={bot.addressKey}
              >
                <span
                  className={`h-2 w-2 shrink-0 rounded-full ${bot.isRunning ? "bg-green-500 animate-pulse" : "bg-muted-foreground/40"}`}
                />
                <span className="text-xs font-mono text-foreground truncate max-w-[140px]">
                  {shortWalletLabel(bot.addressKey)}
                </span>
              </button>
              <div className="flex items-center gap-0.5 border-l border-border/50 pl-1.5 ml-0.5">
                {bot.isRunning ? (
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation();
                      onStop(bot.addressKey);
                    }}
                    className="p-1 rounded hover:bg-destructive/15 text-destructive"
                    title="Stop"
                  >
                    <Square className="w-3.5 h-3.5" />
                  </button>
                ) : (
                  <button
                    type="button"
                    disabled={!backendReady}
                    onClick={(e) => {
                      e.stopPropagation();
                      onStart(bot.addressKey);
                    }}
                    className="p-1 rounded hover:bg-primary/15 text-primary disabled:opacity-40 disabled:pointer-events-none"
                    title="Start"
                  >
                    <Play className="w-3.5 h-3.5" />
                  </button>
                )}
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation();
                    onRemove(bot.addressKey);
                  }}
                  className="p-1 rounded hover:bg-secondary text-muted-foreground hover:text-foreground"
                  title="Remove from list"
                >
                  <X className="w-3.5 h-3.5" />
                </button>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
