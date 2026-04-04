import type { Dispatch, ReactNode, SetStateAction } from "react";
import {
  FileText,
  Loader2,
  Save,
  SlidersHorizontal,
  Wand2,
  X,
} from "lucide-react";
import { cn } from "@/lib/utils";
import type { DescriptionAgentStrategyTarget } from "@/types/description-agent-strategies";
import {
  BTN_GHOST,
  BTN_PRIMARY_ACTION,
  CHECKBOX_FIELD,
  INPUT_FIELD,
  INPUT_READONLY,
  MODAL_BTN_FOCUS,
  PLACEHOLDER_HINT,
  TEXTAREA_FIELD,
  TEXTAREA_MONO,
} from "./constants";
import type { AgentsPanel, ModalTab, StrategyForm } from "./strategyForm";

type ModalProps = {
  panel: AgentsPanel;
  modalTab: ModalTab;
  form: StrategyForm;
  intentDraft: string;
  generating: boolean;
  saving: boolean;
  onClose: () => void;
  setModalTab: (t: ModalTab) => void;
  setIntentDraft: Dispatch<SetStateAction<string>>;
  setForm: Dispatch<SetStateAction<StrategyForm>>;
  toggleTarget: (t: DescriptionAgentStrategyTarget) => void;
  onExpand: () => void | Promise<void>;
  onSubmit: () => void | Promise<void>;
};

function TabButton(props: {
  selected: boolean;
  icon: ReactNode;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      role="tab"
      aria-selected={props.selected}
      onClick={props.onClick}
      className={cn(
        "flex items-center gap-2 px-3 py-2 rounded-t-lg text-sm font-mono transition-colors",
        MODAL_BTN_FOCUS,
        props.selected
          ? "bg-primary/15 text-primary border border-b-0 border-border"
          : "text-muted-foreground hover:text-foreground"
      )}
    >
      {props.icon}
      {props.label}
    </button>
  );
}

function StrategyTabContent(props: {
  intentDraft: string;
  form: StrategyForm;
  onIntentChange: (v: string) => void;
  onToggleTarget: (t: DescriptionAgentStrategyTarget) => void;
}) {
  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">
        Describe what this agent should evaluate in plain language. Targets limit which rows run. An
        LLM will propose the internal id, display name, and prompts—you can tweak them in{" "}
        <span className="text-foreground/90">Advanced</span>.
      </p>
      <label className="block space-y-2">
        <span className="text-xs font-mono text-muted-foreground uppercase tracking-wide">
          What should this agent do?
        </span>
        <textarea
          value={props.intentDraft}
          onChange={(e) => props.onIntentChange(e.target.value)}
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
            <label key={t} className="flex items-center gap-2 text-sm cursor-pointer">
              <input
                type="checkbox"
                checked={props.form.targets.includes(t)}
                onChange={() => props.onToggleTarget(t)}
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
  );
}

function AdvancedTabContent(props: {
  panel: AgentsPanel;
  form: StrategyForm;
  setForm: Dispatch<SetStateAction<StrategyForm>>;
}) {
  return (
    <div className="space-y-4">
      <p className="text-xs text-muted-foreground">
        Technical prompts and id. Targets stay on the Strategy tab. Display name, sort order, and
        model overrides use defaults or values from “Generate prompts” / existing row.
      </p>
      <p className="text-xs text-muted-foreground font-mono">{PLACEHOLDER_HINT}</p>
      {props.panel === "new" ? (
        <label className="block space-y-1">
          <span className="text-xs font-mono text-muted-foreground">
            Strategy id *{" "}
            <span className="text-muted-foreground/70 font-sans">(from AI or type manually)</span>
          </span>
          <input
            value={props.form.id}
            onChange={(e) => props.setForm((f) => ({ ...f, id: e.target.value }))}
            className={`${INPUT_FIELD} font-mono`}
            placeholder="generated after “Generate prompts”"
          />
        </label>
      ) : (
        <label className="block space-y-1">
          <span className="text-xs font-mono text-muted-foreground">Strategy id</span>
          <input value={props.form.id} readOnly className={INPUT_READONLY} />
        </label>
      )}
      <label className="block space-y-1">
        <span className="text-xs font-mono text-muted-foreground">System prompt *</span>
        <textarea
          value={props.form.system_prompt}
          onChange={(e) => props.setForm((f) => ({ ...f, system_prompt: e.target.value }))}
          rows={4}
          className={`${TEXTAREA_MONO} min-h-[88px] max-h-[28vh]`}
        />
      </label>
      <label className="block space-y-1">
        <span className="text-xs font-mono text-muted-foreground">User prompt template *</span>
        <textarea
          value={props.form.user_prompt_template}
          onChange={(e) => props.setForm((f) => ({ ...f, user_prompt_template: e.target.value }))}
          rows={5}
          className={`${TEXTAREA_MONO} min-h-[120px] max-h-[32vh]`}
        />
      </label>
      <label className="flex items-center gap-2 text-sm cursor-pointer w-fit">
        <input
          type="checkbox"
          checked={props.form.enabled}
          onChange={(e) => props.setForm((f) => ({ ...f, enabled: e.target.checked }))}
          className={CHECKBOX_FIELD}
        />
        <span className="text-muted-foreground">Enabled</span>
      </label>
    </div>
  );
}

export function StrategyEditorModal(props: ModalProps) {
  const { panel } = props;
  if (!panel) return null;

  return (
    <div className="fixed inset-0 z-[100] flex items-end sm:items-center justify-center bg-background/80 backdrop-blur-sm p-4">
      <div className="flex h-[min(90vh,720px)] max-h-[90vh] w-full max-w-3xl flex-col rounded-xl border border-border bg-card shadow-xl terminal-border-glow overflow-hidden">
        <div className="flex shrink-0 items-center justify-between px-4 py-3 border-b border-border bg-card/95">
          <h2 className="font-mono text-sm font-semibold text-primary">
            {panel === "new" ? "New strategy" : `Edit ${panel.edit}`}
          </h2>
          <button
            type="button"
            onClick={props.onClose}
            className={`p-2 rounded-lg hover:bg-secondary ${MODAL_BTN_FOCUS}`}
            aria-label="Close"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="flex shrink-0 gap-1 px-4 pt-3 border-b border-border/60">
          <TabButton
            selected={props.modalTab === "strategy"}
            icon={<FileText className="w-4 h-4" />}
            label="Strategy"
            onClick={() => props.setModalTab("strategy")}
          />
          <TabButton
            selected={props.modalTab === "advanced"}
            icon={<SlidersHorizontal className="w-4 h-4" />}
            label="Advanced"
            onClick={() => props.setModalTab("advanced")}
          />
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain p-4">
          {props.modalTab === "strategy" ? (
            <StrategyTabContent
              intentDraft={props.intentDraft}
              form={props.form}
              onIntentChange={props.setIntentDraft}
              onToggleTarget={props.toggleTarget}
            />
          ) : (
            <AdvancedTabContent panel={panel} form={props.form} setForm={props.setForm} />
          )}
        </div>

        <div className="shrink-0 flex flex-wrap items-center justify-between gap-2 px-4 py-3 border-t border-border bg-card/95">
          <button type="button" onClick={props.onClose} className={BTN_GHOST}>
            Cancel
          </button>
          {props.modalTab === "strategy" ? (
            <div className="flex flex-wrap gap-2 justify-end">
              <button
                type="button"
                onClick={() => props.setModalTab("advanced")}
                className={`px-4 py-2 rounded-lg border border-border text-sm text-muted-foreground hover:text-foreground hover:bg-secondary/80 ${MODAL_BTN_FOCUS}`}
              >
                Advanced — manual prompts
              </button>
              <button
                type="button"
                onClick={() => void props.onExpand()}
                disabled={props.generating}
                className={BTN_PRIMARY_ACTION}
              >
                {props.generating ? (
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
                onClick={() => props.setModalTab("strategy")}
                className={BTN_GHOST}
              >
                ← Strategy
              </button>
              <button
                type="button"
                onClick={() => void props.onSubmit()}
                disabled={props.saving}
                className={BTN_PRIMARY_ACTION}
              >
                {props.saving ? (
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
  );
}
