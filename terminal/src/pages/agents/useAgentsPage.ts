import { useCallback, useEffect, useState } from "react";
import type {
  DescriptionAgentStrategy,
  DescriptionAgentStrategyTarget,
} from "@/types/description-agent-strategies";
import {
  deleteStrategy,
  fetchStrategiesList,
  patchStrategy,
  postExpandStrategy,
  postStrategy,
} from "./descriptionAgentStrategiesApi";
import { mergeIntentIntoNotes, extractIntentFromNotes } from "./strategyNotes";
import {
  editIdFromPanel,
  emptyStrategyForm,
  strategyToForm,
  type AgentsPanel,
  type ModalTab,
  type StrategyForm,
} from "./strategyForm";

export function useAgentsPage() {
  const [strategies, setStrategies] = useState<DescriptionAgentStrategy[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [panel, setPanel] = useState<AgentsPanel>(null);
  const [form, setForm] = useState<StrategyForm>(emptyStrategyForm());
  const [modalTab, setModalTab] = useState<ModalTab>("strategy");
  const [intentDraft, setIntentDraft] = useState("");
  const [generating, setGenerating] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    const result = await fetchStrategiesList();
    if (!result.ok) {
      setError(result.error);
      setStrategies([]);
    } else {
      setStrategies(result.strategies);
    }
    setLoading(false);
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const resetModalUi = useCallback(() => {
    setIntentDraft("");
    setModalTab("strategy");
    setGenerating(false);
  }, []);

  const openNew = useCallback(() => {
    setForm(emptyStrategyForm());
    resetModalUi();
    setPanel("new");
  }, [resetModalUi]);

  const openEdit = useCallback((s: DescriptionAgentStrategy) => {
    setForm(strategyToForm(s));
    setIntentDraft(extractIntentFromNotes(s.notes));
    setModalTab("advanced");
    setGenerating(false);
    setPanel({ edit: s.id });
  }, []);

  const closePanel = useCallback(() => {
    setPanel(null);
    resetModalUi();
  }, [resetModalUi]);

  const runExpand = useCallback(async () => {
    const intent = intentDraft.trim();
    if (intent.length < 8) {
      setError("Write a bit more: describe what this agent should judge (8+ characters).");
      return;
    }
    setGenerating(true);
    setError(null);
    const existingId = editIdFromPanel(panel);
    const result = await postExpandStrategy({
      intent,
      targets: form.targets,
      existingId,
    });
    if (!result.ok) {
      setError(result.error);
      setGenerating(false);
      return;
    }
    const g = result.generated;
    setForm((f) => ({
      ...f,
      id: existingId ?? g.id,
      display_name: g.display_name,
      system_prompt: g.system_prompt,
      user_prompt_template: g.user_prompt_template,
      notes: mergeIntentIntoNotes(intent, f.notes),
    }));
    setModalTab("advanced");
    setGenerating(false);
  }, [form.targets, intentDraft, panel]);

  const toggleTarget = useCallback((t: DescriptionAgentStrategyTarget) => {
    setForm((f) => {
      const has = f.targets.includes(t);
      const next = has ? f.targets.filter((x) => x !== t) : [...f.targets, t];
      return { ...f, targets: next };
    });
  }, []);

  const submit = useCallback(async () => {
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
        const result = await postStrategy(body);
        if (!result.ok) {
          setError(result.error);
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
        const result = await patchStrategy(panel.edit, body);
        if (!result.ok) {
          setError(result.error);
          return;
        }
      }
      closePanel();
      await load();
    } finally {
      setSaving(false);
    }
  }, [closePanel, form, load, panel]);

  const remove = useCallback(
    async (id: string) => {
      if (!window.confirm(`Delete strategy "${id}"?`)) return;
      setError(null);
      const result = await deleteStrategy(id);
      if (!result.ok) {
        setError(result.error);
        return;
      }
      if (typeof panel === "object" && panel !== null && panel.edit === id) {
        closePanel();
      }
      await load();
    },
    [closePanel, load, panel]
  );

  return {
    strategies,
    loading,
    error,
    saving,
    panel,
    form,
    modalTab,
    intentDraft,
    generating,
    setError,
    setForm,
    setModalTab,
    setIntentDraft,
    openNew,
    openEdit,
    closePanel,
    runExpand,
    toggleTarget,
    submit,
    remove,
  };
}
