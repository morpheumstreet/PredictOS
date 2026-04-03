import { join } from "path";

/**
 * Path to strat/alpha-rules/data/alpha_rules.sqlite (sibling of terminal/).
 * Override with ALPHA_RULES_DB for custom deployments.
 */
export function getAlphaRulesDbPath(): string {
  const override = process.env.ALPHA_RULES_DB?.trim();
  if (override) return override;
  return join(import.meta.dir, "../../strat/alpha-rules/data/alpha_rules.sqlite");
}
