import type {
  DescriptionAgentStrategy,
  DescriptionAgentStrategyTarget,
} from "@/types/description-agent-strategies";

export type StrategyForm = {
  id: string;
  /** Set by AI / load; not shown in Advanced (table + API still use it). */
  display_name: string;
  targets: DescriptionAgentStrategyTarget[];
  system_prompt: string;
  user_prompt_template: string;
  enabled: boolean;
  /** Intent block + extras; not shown in Advanced. */
  notes: string;
};

export type AgentsPanel = "new" | { edit: string } | null;

export type ModalTab = "strategy" | "advanced";

export function emptyStrategyForm(): StrategyForm {
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

export function strategyToForm(s: DescriptionAgentStrategy): StrategyForm {
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

export function editIdFromPanel(panel: AgentsPanel): string | undefined {
  if (panel !== "new" && panel !== null && typeof panel === "object") {
    return panel.edit;
  }
  return undefined;
}
