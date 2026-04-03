import { useCallback, useEffect, useState } from "react";
import {
  Loader2,
  Plus,
  Pencil,
  Trash2,
  Sparkles,
  X,
  Save,
  AlertTriangle,
  Wand2,
  SlidersHorizontal,
  FileText,
} from "lucide-react";
import { cn } from "@/lib/utils";
import Sidebar from "@/components/Sidebar";
import type {
  DescriptionAgentStrategy,
  DescriptionAgentStrategyTarget,
} from "@/types/description-agent-strategies";

const PLACEHOLDER_HINT = `Placeholders: {event_title} {event_slug} {description} {resolution_source} — markets also: {market_question} {market_slug} {market_id} {event_id}`;

/** Matches Wallet Tracking wallet address field (WalletTrackingTerminal) */
const FIELD_BASE =
  "rounded-lg border border-border bg-secondary/50 text-sm transition-all placeholder:text-muted-foreground/50 hover:border-primary/50 focus:outline-none focus:border-primary disabled:opacity-50 disabled:cursor-not-allowed";
const INPUT_FIELD = `w-full px-4 py-3 font-mono ${FIELD_BASE}`;
const TEXTAREA_FIELD = `w-full px-4 py-3 ${FIELD_BASE} resize-y leading-relaxed`;
const TEXTAREA_MONO = `w-full px-4 py-3 font-mono ${FIELD_BASE} resize-y`;
const INPUT_READONLY =
  "w-full cursor-default rounded-lg border border-border bg-secondary/40 px-4 py-3 font-mono text-sm opacity-90 focus:outline-none";
const CHECKBOX_FIELD =
  "h-4 w-4 cursor-pointer rounded border-border bg-secondary/50 accent-primary focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/50 focus-visible:ring-offset-2 focus-visible:ring-offset-card";

const MODAL_BTN_FOCUS =
  "focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/45 focus-visible:ring-offset-2 focus-visible:ring-offset-card";

const INTENT_NOTE_PREFIX = "Strategy intent:\n";

function extractIntentFromNotes(notes: string | null | undefined): string {
  if (!notes) return "";
  if (notes.startsWith(INTENT_NOTE_PREFIX)) {
    const rest = notes.slice(INTENT_NOTE_PREFIX.length);
    const cut = rest.indexOf("\n\n---\n");
    return (cut >= 0 ? rest.slice(0, cut) : rest).trim();
  }
  return "";
}

function mergeIntentIntoNotes(intent: string, previousNotes: string): string {
  const trimmed = intent.trim();
  let tail = "";
  if (previousNotes.startsWith(INTENT_NOTE_PREFIX)) {
    const idx = previousNotes.indexOf("\n\n---\n");
    if (idx >= 0) {
      tail = previousNotes.slice(idx + "\n\n---\n".length).trim();
    }
  } else if (previousNotes.trim()) {
    tail = previousNotes.trim();
  }
  const block = `${INTENT_NOTE_PREFIX}${trimmed}`;
  if (tail) return `${block}\n\n---\n${tail}`;
  return block;
}

function emptyForm(): {
  id: string;
  /** Set by AI / load; not shown in Advanced (table + API still use it). */
  display_name: string;
  targets: DescriptionAgentStrategyTarget[];
  system_prompt: string;
  user_prompt_template: string;
  enabled: boolean;
  /** Intent block + extras; not shown in Advanced. */
  notes: string;
} {
  return {
    id: "",
    display_name: "",
    targets: [],
    system_prompt: "",
    user_prompt_template: "",
    enabled: true,
    notes: "",
  };
}

function strategyToForm(s: DescriptionAgentStrategy) {
  return {
    id: s.id,
    display_name: s.display_name ?? "",
    targets: [...s.targets] as DescriptionAgentStrategyTarget[],
    system_prompt: s.system_prompt,
    user_prompt_template: s.user_prompt_template,
    enabled: s.enabled,
    notes: s.notes ?? "",
  };
}

export function AgentsPage() {
  const [strategies, setStrategies] = useState<DescriptionAgentStrategy[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [panel, setPanel] = useState<"new" | { edit: string } | null>(null);
  const [form, setForm] = useState(emptyForm());
  const [modalTab, setModalTab] = useState<"strategy" | "advanced">("strategy");
  const [intentDraft, setIntentDraft] = useState("");
  const [generating, setGenerating] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/description-agent-strategies");
      const data = (await res.json()) as {
        success?: boolean;
        strategies?: DescriptionAgentStrategy[];
        error?: string;
      };
      if (!res.ok || !data.success || !Array.isArray(data.strategies)) {
        setError(data.error || `Request failed (${res.status})`);
        setStrategies([]);
        return;
      }
      setStrategies(data.strategies);
    } catch {
      setError("Network error loading strategies");
      setStrategies([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const openNew = () => {
    setForm(emptyForm());
    setIntentDraft("");
    setModalTab("strategy");
    setGenerating(false);
    setPanel("new");
  };

  const openEdit = (s: DescriptionAgentStrategy) => {
    setForm(strategyToForm(s));
    setIntentDraft(extractIntentFromNotes(s.notes));
    setModalTab("advanced");
    setGenerating(false);
    setPanel({ edit: s.id });
  };

  const closePanel = () => {
    setPanel(null);
    setIntentDraft("");
    setModalTab("strategy");
    setGenerating(false);
  };

  const runExpand = async () => {
    const intent = intentDraft.trim();
    if (intent.length < 8) {
      setError("Write a bit more: describe what this agent should judge (8+ characters).");
      return;
    }
    setGenerating(true);
    setError(null);
    try {
      const existingId =
        panel !== "new" && typeof panel === "object" && panel !== null
          ? panel.edit
          : undefined;
      const res = await fetch("/api/description-agent-strategies-expand", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          intent,
          targets: form.targets,
          existing_id: existingId,
        }),
      });
      const data = (await res.json()) as {
        success?: boolean;
        error?: string;
        generated?: {
          id: string;
          display_name: string;
          system_prompt: string;
          user_prompt_template: string;
        };
      };
      if (!res.ok || !data.success || !data.generated) {
        setError(data.error || `Generate failed (${res.status})`);
        return;
      }
      const g = data.generated;
      setForm((f) => ({
        ...f,
        id: existingId ?? g.id,
        display_name: g.display_name,
        system_prompt: g.system_prompt,
        user_prompt_template: g.user_prompt_template,
        notes: mergeIntentIntoNotes(intent, f.notes),
      }));
      setModalTab("advanced");
    } catch {
      setError("Network error while generating prompts");
    } finally {
      setGenerating(false);
    }
  };

  const toggleTarget = (t: DescriptionAgentStrategyTarget) => {
    setForm((f) => {
      const has = f.targets.includes(t);
      const next = has ? f.targets.filter((x) => x !== t) : [...f.targets, t];
      return { ...f, targets: next };
    });
  };

  const submit = async () => {
    if (
      panel === "new" &&
      (!form.id.trim() || !form.system_prompt.trim() || !form.user_prompt_template.trim())
    ) {
      setError(
        "Generate prompts on the Strategy tab, or switch to Advanced and fill id, system, and user template."
      );
      setModalTab("advanced");
      return;
    }
    setSaving(true);
    setError(null);
    try {
      if (panel === "new") {
        const body: Record<string, unknown> = {
          id: form.id.trim(),
          display_name: form.display_name.trim() || null,
          targets: form.targets,
          system_prompt: form.system_prompt,
          user_prompt_template: form.user_prompt_template,
          enabled: form.enabled,
          sort_order: 0,
          model: null,
          temperature: null,
          notes: form.notes.trim() || null,
        };

        const res = await fetch("/api/description-agent-strategies", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        });
        const data = (await res.json()) as { success?: boolean; error?: string };
        if (!res.ok || !data.success) {
          setError(data.error || `Create failed (${res.status})`);
          return;
        }
      } else if (panel && "edit" in panel) {
        const body: Record<string, unknown> = {
          display_name: form.display_name.trim() || null,
          targets: form.targets,
          system_prompt: form.system_prompt,
          user_prompt_template: form.user_prompt_template,
          enabled: form.enabled,
        };

        const res = await fetch(
          `/api/description-agent-strategies?id=${encodeURIComponent(panel.edit)}`,
          {
            method: "PATCH",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(body),
          }
        );
        const data = (await res.json()) as { success?: boolean; error?: string };
        if (!res.ok || !data.success) {
          setError(data.error || `Update failed (${res.status})`);
          return;
        }
      }
      closePanel();
      await load();
    } finally {
      setSaving(false);
    }
  };

  const remove = async (id: string) => {
    if (!window.confirm(`Delete strategy "${id}"?`)) return;
    setError(null);
    const res = await fetch(
      `/api/description-agent-strategies?id=${encodeURIComponent(id)}`,
      { method: "DELETE" }
    );
    const data = (await res.json()) as { success?: boolean; error?: string };
    if (!res.ok || !data.success) {
      setError(data.error || `Delete failed (${res.status})`);
      return;
    }
    if (typeof panel === "object" && panel !== null && panel.edit === id) {
      closePanel();
    }
    await load();
  };

  return (
    <div className="flex h-screen">
      <div className="relative z-10 overflow-visible">
        <Sidebar activeTab="agents" />
      </div>
      <main className="flex-1 overflow-y-auto overflow-x-hidden p-6 space-y-6">
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
          <button
            type="button"
            onClick={openNew}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg bg-primary/15 border border-primary/40 text-primary hover:bg-primary/25 transition-colors text-sm font-mono ${MODAL_BTN_FOCUS}`}
          >
            <Plus className="w-4 h-4" />
            New strategy
          </button>
        </header>

        {error && (
          <div className="flex items-start gap-2 p-3 rounded-lg border border-destructive/50 bg-destructive/10 text-sm text-destructive">
            <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5" />
            <span>{error}</span>
          </div>
        )}

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
                      <td className="p-3 text-foreground/90">
                        {s.display_name || "—"}
                      </td>
                      <td className="p-3 font-mono text-xs text-muted-foreground">
                        {s.targets.length ? s.targets.join(", ") : "events, markets"}
                      </td>
                      <td className="p-3">{s.enabled ? "yes" : "no"}</td>
                      <td className="p-3 font-mono">{s.sort_order}</td>
                      <td className="p-3 font-mono text-xs text-muted-foreground">
                        {s.updated_at}
                      </td>
                      <td className="p-3">
                        <div className="flex gap-1">
                          <button
                            type="button"
                            onClick={() => openEdit(s)}
                            className={`p-2 rounded-md hover:bg-secondary border border-transparent hover:border-border ${MODAL_BTN_FOCUS}`}
                            title="Edit"
                          >
                            <Pencil className="w-4 h-4" />
                          </button>
                          <button
                            type="button"
                            onClick={() => void remove(s.id)}
                            className={`p-2 rounded-md hover:bg-destructive/15 border border-transparent hover:border-destructive/40 text-destructive ${MODAL_BTN_FOCUS}`}
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

        {panel && (
          <div className="fixed inset-0 z-[100] flex items-end sm:items-center justify-center bg-background/80 backdrop-blur-sm p-4">
            <div className="flex h-[min(90vh,720px)] max-h-[90vh] w-full max-w-3xl flex-col rounded-xl border border-border bg-card shadow-xl terminal-border-glow overflow-hidden">
              <div className="flex shrink-0 items-center justify-between px-4 py-3 border-b border-border bg-card/95">
                <h2 className="font-mono text-sm font-semibold text-primary">
                  {panel === "new" ? "New strategy" : `Edit ${panel.edit}`}
                </h2>
                <button
                  type="button"
                  onClick={closePanel}
                  className={`p-2 rounded-lg hover:bg-secondary ${MODAL_BTN_FOCUS}`}
                  aria-label="Close"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>

              <div className="flex shrink-0 gap-1 px-4 pt-3 border-b border-border/60">
                <button
                  type="button"
                  role="tab"
                  aria-selected={modalTab === "strategy"}
                  onClick={() => setModalTab("strategy")}
                  className={cn(
                    "flex items-center gap-2 px-3 py-2 rounded-t-lg text-sm font-mono transition-colors",
                    MODAL_BTN_FOCUS,
                    modalTab === "strategy"
                      ? "bg-primary/15 text-primary border border-b-0 border-border"
                      : "text-muted-foreground hover:text-foreground"
                  )}
                >
                  <FileText className="w-4 h-4" />
                  Strategy
                </button>
                <button
                  type="button"
                  role="tab"
                  aria-selected={modalTab === "advanced"}
                  onClick={() => setModalTab("advanced")}
                  className={cn(
                    "flex items-center gap-2 px-3 py-2 rounded-t-lg text-sm font-mono transition-colors",
                    MODAL_BTN_FOCUS,
                    modalTab === "advanced"
                      ? "bg-primary/15 text-primary border border-b-0 border-border"
                      : "text-muted-foreground hover:text-foreground"
                  )}
                >
                  <SlidersHorizontal className="w-4 h-4" />
                  Advanced
                </button>
              </div>

              <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain p-4">
                {modalTab === "strategy" ? (
                  <div className="space-y-4">
                    <p className="text-sm text-muted-foreground">
                      Describe what this agent should evaluate in plain language. Targets limit which
                      rows run. An LLM will propose the internal id, display name, and prompts—you can
                      tweak them in <span className="text-foreground/90">Advanced</span>.
                    </p>
                    <label className="block space-y-2">
                      <span className="text-xs font-mono text-muted-foreground uppercase tracking-wide">
                        What should this agent do?
                      </span>
                      <textarea
                        value={intentDraft}
                        onChange={(e) => setIntentDraft(e.target.value)}
                        rows={6}
                        className={`${TEXTAREA_FIELD} min-h-[140px] max-h-[40vh]`}
                        placeholder="Example: Flag markets where the resolution rules rely on subjective wording or undefined data sources, so we can review them manually."
                      />
                    </label>
                    <div className="space-y-2">
                      <span className="text-xs font-mono text-muted-foreground uppercase tracking-wide">
                        Targets
                      </span>
                      <div className="flex flex-wrap gap-4">
                        {(["events", "markets"] as const).map((t) => (
                          <label
                            key={t}
                            className="flex items-center gap-2 text-sm cursor-pointer"
                          >
                            <input
                              type="checkbox"
                              checked={form.targets.includes(t)}
                              onChange={() => toggleTarget(t)}
                              className={CHECKBOX_FIELD}
                            />
                            {t}
                          </label>
                        ))}
                      </div>
                      <p className="text-[11px] text-muted-foreground">
                        Leave both unchecked to include events and markets.
                      </p>
                    </div>
                    <p className="text-xs text-muted-foreground">
                      Requires <code className="text-primary/90">OPENAI_API_KEY</code> in{" "}
                      <code className="text-primary/90">terminal/.env</code>. Optional:{" "}
                      <code className="text-primary/90">DESCRIPTION_AGENT_EXPAND_MODEL</code>.
                    </p>
                  </div>
                ) : (
                  <div className="space-y-4">
                    <p className="text-xs text-muted-foreground">
                      Technical prompts and id. Targets stay on the Strategy tab. Display name, sort
                      order, and model overrides use defaults or values from “Generate prompts” /
                      existing row.
                    </p>
                    <p className="text-xs text-muted-foreground font-mono">{PLACEHOLDER_HINT}</p>
                    {panel === "new" ? (
                      <label className="block space-y-1">
                        <span className="text-xs font-mono text-muted-foreground">
                          Strategy id *{" "}
                          <span className="text-muted-foreground/70 font-sans">
                            (from AI or type manually)
                          </span>
                        </span>
                        <input
                          value={form.id}
                          onChange={(e) => setForm((f) => ({ ...f, id: e.target.value }))}
                          className={`${INPUT_FIELD} font-mono`}
                          placeholder="generated after “Generate prompts”"
                        />
                      </label>
                    ) : (
                      <label className="block space-y-1">
                        <span className="text-xs font-mono text-muted-foreground">Strategy id</span>
                        <input
                          value={form.id}
                          readOnly
                          className={INPUT_READONLY}
                        />
                      </label>
                    )}
                    <label className="block space-y-1">
                      <span className="text-xs font-mono text-muted-foreground">System prompt *</span>
                      <textarea
                        value={form.system_prompt}
                        onChange={(e) =>
                          setForm((f) => ({ ...f, system_prompt: e.target.value }))
                        }
                        rows={4}
                        className={`${TEXTAREA_MONO} min-h-[88px] max-h-[28vh]`}
                      />
                    </label>
                    <label className="block space-y-1">
                      <span className="text-xs font-mono text-muted-foreground">
                        User prompt template *
                      </span>
                      <textarea
                        value={form.user_prompt_template}
                        onChange={(e) =>
                          setForm((f) => ({ ...f, user_prompt_template: e.target.value }))
                        }
                        rows={5}
                        className={`${TEXTAREA_MONO} min-h-[120px] max-h-[32vh]`}
                      />
                    </label>
                    <label className="flex items-center gap-2 text-sm cursor-pointer w-fit">
                      <input
                        type="checkbox"
                        checked={form.enabled}
                        onChange={(e) =>
                          setForm((f) => ({ ...f, enabled: e.target.checked }))
                        }
                        className={CHECKBOX_FIELD}
                      />
                      <span className="text-muted-foreground">Enabled</span>
                    </label>
                  </div>
                )}
              </div>

              <div className="shrink-0 flex flex-wrap items-center justify-between gap-2 px-4 py-3 border-t border-border bg-card/95">
                <button
                  type="button"
                  onClick={closePanel}
                  className={`px-4 py-2 rounded-lg border border-border text-sm hover:bg-secondary ${MODAL_BTN_FOCUS}`}
                >
                  Cancel
                </button>
                {modalTab === "strategy" ? (
                  <div className="flex flex-wrap gap-2 justify-end">
                    <button
                      type="button"
                      onClick={() => setModalTab("advanced")}
                      className={`px-4 py-2 rounded-lg border border-border text-sm text-muted-foreground hover:text-foreground hover:bg-secondary/80 ${MODAL_BTN_FOCUS}`}
                    >
                      Advanced — manual prompts
                    </button>
                    <button
                      type="button"
                      onClick={() => void runExpand()}
                      disabled={generating}
                      className={`flex items-center gap-2 px-4 py-2 rounded-lg bg-primary/20 border border-primary/50 text-primary text-sm font-mono hover:bg-primary/30 disabled:opacity-50 disabled:focus-visible:ring-0 ${MODAL_BTN_FOCUS}`}
                    >
                      {generating ? (
                        <Loader2 className="w-4 h-4 animate-spin" />
                      ) : (
                        <Wand2 className="w-4 h-4" />
                      )}
                      {panel === "new" ? "Generate prompts" : "Regenerate prompts"}
                    </button>
                  </div>
                ) : (
                  <div className="flex flex-wrap gap-2 justify-end">
                    <button
                      type="button"
                      onClick={() => setModalTab("strategy")}
                      className={`px-4 py-2 rounded-lg border border-border text-sm hover:bg-secondary ${MODAL_BTN_FOCUS}`}
                    >
                      ← Strategy
                    </button>
                    <button
                      type="button"
                      onClick={() => void submit()}
                      disabled={saving}
                      className={`flex items-center gap-2 px-4 py-2 rounded-lg bg-primary/20 border border-primary/50 text-primary text-sm font-mono hover:bg-primary/30 disabled:opacity-50 disabled:focus-visible:ring-0 ${MODAL_BTN_FOCUS}`}
                    >
                      {saving ? (
                        <Loader2 className="w-4 h-4 animate-spin" />
                      ) : (
                        <Save className="w-4 h-4" />
                      )}
                      Save
                    </button>
                  </div>
                )}
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}
