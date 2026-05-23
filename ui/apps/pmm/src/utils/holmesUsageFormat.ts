export interface HolmesUsageDisplay {
  model?: string;
  totalTokens?: number;
  cachedTokens?: number;
  totalCost?: number;
  promptTokens?: number;
  completionTokens?: number;
}

export function formatTokenCount(n: number | undefined): string {
  if (n == null || Number.isNaN(n)) return '';
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}k`;
  return String(n);
}

export function formatUsdCost(cost: number | undefined): string {
  if (cost == null || Number.isNaN(cost)) return '';
  if (cost >= 1) return `$${cost.toFixed(2)}`;
  if (cost >= 0.01) return `$${cost.toFixed(3)}`;
  return `$${cost.toFixed(4)}`;
}

export function holmesUsageSummaryLine(usage: HolmesUsageDisplay): string | null {
  const parts: string[] = [];
  if (usage.model) parts.push(usage.model);
  if (usage.totalTokens != null) {
    let tok = `${formatTokenCount(usage.totalTokens)} tokens`;
    if (usage.cachedTokens != null && usage.cachedTokens > 0) {
      tok += ` (${formatTokenCount(usage.cachedTokens)} cached)`;
    }
    parts.push(tok);
  }
  const cost = formatUsdCost(usage.totalCost);
  if (cost) parts.push(cost);
  return parts.length ? parts.join(' · ') : null;
}

export function holmesUsageTooltip(usage: HolmesUsageDisplay): string {
  const lines: string[] = [];
  if (usage.promptTokens != null) lines.push(`Prompt: ${usage.promptTokens.toLocaleString()}`);
  if (usage.completionTokens != null) lines.push(`Completion: ${usage.completionTokens.toLocaleString()}`);
  if (usage.cachedTokens != null) lines.push(`Cached: ${usage.cachedTokens.toLocaleString()}`);
  if (usage.totalCost != null) lines.push(`Cost: ${formatUsdCost(usage.totalCost)}`);
  return lines.join('\n');
}

export const HOLMES_FEATURE_LABELS: Record<string, string> = {
  adre_chat: 'ADRE chat',
  investigation_chat: 'Investigation chat',
  investigation_run: 'Full report',
  investigation_format: 'Format report',
  qan_insights: 'QAN insights',
  slack_chat: 'Slack',
};
