// Grafana ruler API response for a folder's evaluation groups.
// GET /graph/api/ruler/grafana/api/v1/rules/{folderUid}?subtype=cortex
export interface RulerRuleGroup {
  name: string;
  interval?: string;
  rules?: unknown[];
}

export type RulerRulesConfig = Record<string, RulerRuleGroup[]>;
