import type { GrafanaAlertRuleDefinition } from 'types/alerting.types';

export interface RuleExpressionProps {
  definition?: GrafanaAlertRuleDefinition;
  // Grafana's one-line summary, shown until the rule definition has loaded.
  fallback: string;
}
