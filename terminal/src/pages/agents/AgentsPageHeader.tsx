import { Plus, Sparkles } from "lucide-react";
import { BTN_HEADER_NEW } from "./constants";

type Props = { onNew: () => void };

export function AgentsPageHeader({ onNew }: Props) {
  return (
    <header className="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 className="font-display text-2xl font-bold text-foreground flex items-center gap-2">
          <Sparkles className="w-7 h-7 text-primary" />
          Description agents
        </h1>
        <p className="text-sm text-muted-foreground mt-1 max-w-2xl">
          New strategies start on the Strategy tab (plain-language intent + targets); AI drafts id,
          name, and prompts, then you tune on Advanced. Stored in{" "}
          <code className="text-xs text-primary/90">description_agent_strategies</code>; cron{" "}
          <code className="text-xs">description_agent.py</code> picks them up when{" "}
          <code className="text-xs">strategies_source</code> is <code className="text-xs">auto</code> or{" "}
          <code className="text-xs">db</code>.
        </p>
      </div>
      <button type="button" onClick={onNew} className={BTN_HEADER_NEW}>
        <Plus className="w-4 h-4" />
        New strategy
      </button>
    </header>
  );
}
