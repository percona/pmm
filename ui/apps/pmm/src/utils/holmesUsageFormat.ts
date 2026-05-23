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

export interface DailyCostPoint {
  bucket: string;
  totalCost: number;
  totalTokens: number;
  callCount: number;
}

/** Merge API series rows that share the same day bucket (defensive if grouped by day+feature). */
export function aggregateUsageSeriesByDay(
  series: Array<{
    bucket?: string;
    totalCost?: number;
    total_cost?: number;
    totalTokens?: number;
    total_tokens?: number;
    callCount?: number;
    call_count?: number;
  }>
): DailyCostPoint[] {
  const byDay = new Map<string, DailyCostPoint>();
  for (const row of series) {
    const bucket = row.bucket?.trim();
    if (!bucket) continue;
    const cost = row.totalCost ?? row.total_cost ?? 0;
    const tokens = row.totalTokens ?? row.total_tokens ?? 0;
    const calls = row.callCount ?? row.call_count ?? 0;
    const prev = byDay.get(bucket);
    if (prev) {
      byDay.set(bucket, {
        bucket,
        totalCost: prev.totalCost + cost,
        totalTokens: prev.totalTokens + tokens,
        callCount: prev.callCount + calls,
      });
    } else {
      byDay.set(bucket, { bucket, totalCost: cost, totalTokens: tokens, callCount: calls });
    }
  }
  return [...byDay.values()].sort((a, b) => a.bucket.localeCompare(b.bucket));
}

/** One row per calendar day from first usage through [from, to], newest first. */
export function fillDailyCostSeries(
  series: DailyCostPoint[],
  fromISO: string,
  toISO: string
): DailyCostPoint[] {
  const byDay = new Map(series.map((row) => [row.bucket, row]));
  const start = new Date(fromISO);
  const end = new Date(toISO);
  if (Number.isNaN(start.getTime()) || Number.isNaN(end.getTime())) {
    return [...series].sort((a, b) => b.bucket.localeCompare(a.bucket));
  }
  start.setUTCHours(0, 0, 0, 0);
  end.setUTCHours(0, 0, 0, 0);
  const out: DailyCostPoint[] = [];
  for (let t = start.getTime(); t <= end.getTime(); t += 86400000) {
    const bucket = new Date(t).toISOString().slice(0, 10);
    out.push(
      byDay.get(bucket) ?? {
        bucket,
        totalCost: 0,
        totalTokens: 0,
        callCount: 0,
      }
    );
  }
  let first = 0;
  while (first < out.length - 1 && out[first].totalCost === 0) {
    first++;
  }
  return out.slice(first).reverse();
}

export function formatUsageDayLabel(bucket: string): string {
  const d = new Date(`${bucket}T12:00:00Z`);
  if (Number.isNaN(d.getTime())) return bucket;
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
}

export interface InvestigationUsageStep {
  id: string | number;
  createdAt: string;
  feature: string;
  model: string;
  totalTokens?: number;
  cachedTokens?: number;
  totalCost?: number;
}

export interface InvestigationUsageSummary {
  callCount: number;
  totalTokens: number;
  totalCached: number;
  totalCost: number;
  hasUsage: boolean;
  steps: InvestigationUsageStep[];
}

type UsageMessageRow = {
  id?: string;
  role?: string;
  createdAt?: string;
  created_at?: string;
  model?: string;
  holmesFeature?: string;
  holmes_feature?: string;
  totalTokens?: number;
  total_tokens?: number;
  cachedTokens?: number;
  cached_tokens?: number;
  totalCost?: number;
  total_cost?: number;
};

type UsageEventRow = {
  id?: number;
  createdAt?: string;
  created_at?: string;
  feature?: string;
  model?: string;
  totalTokens?: number;
  total_tokens?: number;
  cachedTokens?: number;
  cached_tokens?: number;
  totalCost?: number;
  total_cost?: number;
};

function num(v: number | undefined, fallback = 0): number {
  return v ?? fallback;
}

function maxNum(...values: number[]): number {
  return values.reduce((max, v) => (v > max ? v : max), 0);
}

function messageHasUsage(m: UsageMessageRow): boolean {
  if (m.role !== 'assistant') return false;
  const tokens = m.totalTokens ?? m.total_tokens;
  const cost = m.totalCost ?? m.total_cost ?? 0;
  const model = m.model?.trim();
  return tokens != null || cost > 0 || !!model;
}

function usageStepsFromMessages(messages: UsageMessageRow[]): InvestigationUsageStep[] {
  return messages
    .filter(messageHasUsage)
    .map((m) => ({
      id: m.id ?? `${m.createdAt ?? m.created_at ?? ''}-${m.holmesFeature ?? m.holmes_feature ?? 'assistant'}`,
      createdAt: m.createdAt ?? m.created_at ?? '',
      feature: m.holmesFeature ?? m.holmes_feature ?? 'investigation_chat',
      model: m.model ?? 'default',
      totalTokens: m.totalTokens ?? m.total_tokens,
      cachedTokens: m.cachedTokens ?? m.cached_tokens,
      totalCost: m.totalCost ?? m.total_cost,
    }))
    .sort((a, b) => a.createdAt.localeCompare(b.createdAt));
}

function normalizeUsageEvents(events: unknown): InvestigationUsageStep[] {
  if (!Array.isArray(events)) return [];
  return events
    .map((row) => {
      const ev = row as UsageEventRow;
      return {
        id: ev.id ?? `${ev.createdAt ?? ev.created_at ?? ''}-${ev.feature ?? 'usage'}`,
        createdAt: ev.createdAt ?? ev.created_at ?? '',
        feature: ev.feature ?? 'investigation_run',
        model: ev.model ?? 'default',
        totalTokens: ev.totalTokens ?? ev.total_tokens,
        cachedTokens: ev.cachedTokens ?? ev.cached_tokens,
        totalCost: ev.totalCost ?? ev.total_cost,
      };
    })
    .sort((a, b) => a.createdAt.localeCompare(b.createdAt));
}

function sumSteps(steps: InvestigationUsageStep[]) {
  return steps.reduce(
    (acc, step) => ({
      callCount: acc.callCount + 1,
      totalTokens: acc.totalTokens + num(step.totalTokens),
      totalCached: acc.totalCached + num(step.cachedTokens),
      totalCost: acc.totalCost + num(step.totalCost),
    }),
    { callCount: 0, totalTokens: 0, totalCached: 0, totalCost: 0 }
  );
}

function mergeUsageSteps(
  apiSteps: InvestigationUsageStep[],
  messageSteps: InvestigationUsageStep[]
): InvestigationUsageStep[] {
  const merged = [...apiSteps];
  for (const msgStep of messageSteps) {
    const duplicate = merged.some(
      (apiStep) =>
        apiStep.feature === msgStep.feature &&
        apiStep.createdAt === msgStep.createdAt &&
        num(apiStep.totalTokens) === num(msgStep.totalTokens) &&
        num(apiStep.totalCost) === num(msgStep.totalCost)
    );
    if (!duplicate) {
      merged.push(msgStep);
    }
  }
  return merged.sort((a, b) => a.createdAt.localeCompare(b.createdAt));
}

/** Merge investigation totals, chat messages, and usage events into one display model. */
export function aggregateInvestigationUsage(input: {
  holmesCallCount?: number;
  holmes_call_count?: number;
  holmesTotalTokens?: number;
  holmes_total_tokens?: number;
  holmesTotalCost?: number;
  holmes_total_cost?: number;
  messages?: UsageMessageRow[];
  events?: unknown;
}): InvestigationUsageSummary {
  const messageUsage = aggregateAssistantMessageUsage(input.messages ?? []);
  const apiSteps = normalizeUsageEvents(input.events);
  const messageSteps = usageStepsFromMessages(input.messages ?? []);
  const steps = mergeUsageSteps(apiSteps, messageSteps);
  const stepTotals = sumSteps(steps);

  const callCount = maxNum(
    num(input.holmesCallCount ?? input.holmes_call_count),
    messageUsage.callCount,
    stepTotals.callCount
  );
  const totalTokens = maxNum(
    num(input.holmesTotalTokens ?? input.holmes_total_tokens),
    messageUsage.totalTokens,
    stepTotals.totalTokens
  );
  const totalCached = maxNum(stepTotals.totalCached, messageUsage.totalCached);
  const totalCost = maxNum(
    num(input.holmesTotalCost ?? input.holmes_total_cost),
    messageUsage.totalCost,
    stepTotals.totalCost
  );

  const hasUsage = callCount > 0 || totalTokens > 0 || totalCost > 0 || steps.length > 0;

  return { callCount, totalTokens, totalCached, totalCost, hasUsage, steps };
}
