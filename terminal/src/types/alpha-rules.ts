export interface AlphaRulesSummary {
  success: boolean;
  path?: string;
  exists?: boolean;
  counts?: Record<string, number>;
  lastScanRun?: {
    id: number;
    started_at: string;
    finished_at: string | null;
    status: string;
    events_scanned: number | null;
    error_message: string | null;
  } | null;
  error?: string;
  hint?: string;
}

export interface AlphaRulesEventRow {
  id: string;
  slug: string | null;
  ticker: string | null;
  title: string | null;
  description: string | null;
  resolution_source: string | null;
  start_date: string | null;
  end_date: string | null;
  active: number;
  closed: number;
  volume: number | null;
  liquidity: number | null;
  tags_json: string | null;
  updated_at_api: string | null;
  fetched_at: string;
  external_truth_source_urls: string | null;
  has_profit_opportunity: number;
  last_scanned_at: string | null;
}

export interface AlphaRulesEventsAllResponse {
  success: boolean;
  table: string;
  total: number;
  returned: number;
  truncated: boolean;
  rows: AlphaRulesEventRow[];
  error?: string;
}
