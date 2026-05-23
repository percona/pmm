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

export function formatTokensWithCached(total?: number, cached?: number): string {
  const totalStr = formatTokenCount(total);
  if (!totalStr) return '';
  if (cached != null && cached > 0) {
    return `${totalStr} tokens (${formatTokenCount(cached)} cached)`;
  }
  return `${totalStr} tokens`;
}

export function holmesUsageSummaryLine(usage: HolmesUsageDisplay): string | null {
  const parts: string[] = [];
  if (usage.model) parts.push(usage.model);
  if (usage.totalTokens != null) {
    parts.push(formatTokensWithCached(usage.totalTokens, usage.cachedTokens));
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

export function aggregateAssistantMessageUsage(
  messages: Array<{
    role?: string;
    totalTokens?: number;
    total_tokens?: number;
    cachedTokens?: number;
    cached_tokens?: number;
    totalCost?: number;
    total_cost?: number;
    model?: string;
  }>
): { callCount: number; totalTokens: number; totalCached: number; totalCost: number } {
  let callCount = 0;
  let totalTokens = 0;
  let totalCached = 0;
  let totalCost = 0;
  for (const m of messages) {
    if (m.role !== 'assistant') continue;
    const tokens = m.totalTokens ?? m.total_tokens;
    const cached = m.cachedTokens ?? m.cached_tokens ?? 0;
    const cost = m.totalCost ?? m.total_cost ?? 0;
    const hasUsage = tokens != null || cost > 0 || (m.model != null && m.model !== '');
    if (!hasUsage) continue;
    callCount++;
    totalTokens += tokens ?? 0;
    totalCached += cached;
    totalCost += cost;
  }
  return { callCount, totalTokens, totalCached, totalCost };
}
