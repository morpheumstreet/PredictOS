import { Loader2, Pencil, Trash2 } from "lucide-react";
import type { DescriptionAgentStrategy } from "@/types/description-agent-strategies";
import { BTN_ICON_DELETE, BTN_ICON_EDIT } from "./constants";

type Props = {
  strategies: DescriptionAgentStrategy[];
  loading: boolean;
  onEdit: (s: DescriptionAgentStrategy) => void;
  onDelete: (id: string) => void;
};

export function StrategiesTable({ strategies, loading, onEdit, onDelete }: Props) {
  return (
    <div className="rounded-xl border border-border bg-card/40 overflow-hidden">
      {loading ? (
        <div className="flex items-center justify-center gap-2 py-16 text-muted-foreground">
          <Loader2 className="w-5 h-5 animate-spin" />
          Loading strategies…
        </div>
      ) : strategies.length === 0 ? (
        <div className="py-16 text-center text-muted-foreground text-sm">
          No strategies yet. Create one to drive{" "}
          <code className="text-xs text-foreground/80">description_agent.py</code>.
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-secondary/30 text-left text-muted-foreground font-mono text-xs uppercase tracking-wide">
                <th className="p-3">Id</th>
                <th className="p-3">Name</th>
                <th className="p-3">Targets</th>
                <th className="p-3">On</th>
                <th className="p-3">Order</th>
                <th className="p-3">Updated</th>
                <th className="p-3 w-28">Actions</th>
              </tr>
            </thead>
            <tbody>
              {strategies.map((s) => (
                <tr
                  key={s.id}
                  className="border-b border-border/60 hover:bg-secondary/20 transition-colors"
                >
                  <td className="p-3 font-mono text-primary">{s.id}</td>
                  <td className="p-3 text-foreground/90">{s.display_name || "—"}</td>
                  <td className="p-3 font-mono text-xs text-muted-foreground">
                    {s.targets.length ? s.targets.join(", ") : "events, markets"}
                  </td>
                  <td className="p-3">{s.enabled ? "yes" : "no"}</td>
                  <td className="p-3 font-mono">{s.sort_order}</td>
                  <td className="p-3 font-mono text-xs text-muted-foreground">{s.updated_at}</td>
                  <td className="p-3">
                    <div className="flex gap-1">
                      <button
                        type="button"
                        onClick={() => onEdit(s)}
                        className={BTN_ICON_EDIT}
                        title="Edit"
                      >
                        <Pencil className="w-4 h-4" />
                      </button>
                      <button
                        type="button"
                        onClick={() => void onDelete(s.id)}
                        className={BTN_ICON_DELETE}
                        title="Delete"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
