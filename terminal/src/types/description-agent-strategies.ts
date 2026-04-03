export type DescriptionAgentStrategyTarget = "events" | "markets";

export type DescriptionAgentStrategy = {
  id: string;
  display_name: string | null;
  targets_json: string;
  targets: DescriptionAgentStrategyTarget[];
  system_prompt: string;
  user_prompt_template: string;
  enabled: boolean;
  sort_order: number;
  model: string | null;
  temperature: number | null;
  json_response_format: boolean | null;
  notes: string | null;
  created_at: string;
  updated_at: string;
};

export type DescriptionAgentStrategiesListResponse = {
  success: boolean;
  strategies?: DescriptionAgentStrategy[];
  error?: string;
};

export type DescriptionAgentStrategyMutationResponse = {
  success: boolean;
  strategy?: DescriptionAgentStrategy;
  error?: string;
  deleted?: string;
};
