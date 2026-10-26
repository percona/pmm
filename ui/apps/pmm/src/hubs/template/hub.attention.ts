import { matchesEnvironment } from './hub.params';
import type {
  HubAttentionItem,
  HubAttentionSeverity,
  HubNeedsAttentionItem,
  HubSeverity,
} from './hub.types';

export const HUB_SEVERITY_RANK: Record<HubSeverity, number> = {
  error: 0,
  warning: 1,
  info: 2,
};

/** The more severe of two severities. */
export const higherSeverity = (a: HubSeverity, b: HubSeverity): HubSeverity =>
  HUB_SEVERITY_RANK[a] <= HUB_SEVERITY_RANK[b] ? a : b;

const sinceMs = (item: HubAttentionItem) => {
  const time = item.since ? Date.parse(item.since) : NaN;

  // Items without a start sort after the dated ones of their kind.
  return Number.isNaN(time) ? Number.POSITIVE_INFINITY : time;
};

const kindRank = (order: string[], kind: string) => {
  const index = order.indexOf(kind);

  return index === -1 ? order.length : index;
};

/** Severity first, then the technology's kind order, then the longest-broken, then id. */
export const rankAttention = (
  items: HubAttentionItem[],
  order: string[]
): HubAttentionItem[] =>
  [...items].sort(
    (a, b) =>
      HUB_SEVERITY_RANK[a.severity] - HUB_SEVERITY_RANK[b.severity] ||
      kindRank(order, a.kind) - kindRank(order, b.kind) ||
      sinceMs(a) - sinceMs(b) ||
      (a.id < b.id ? -1 : a.id > b.id ? 1 : 0)
  );

const NEEDS_ATTENTION_RANK: Record<HubAttentionSeverity, number> = {
  critical: 0,
  error: 1,
  warning: 2,
};

/** Worst first; rows of one severity keep their order, which is by impact, then longest first. */
export const sortNeedsAttention = (
  items: HubNeedsAttentionItem[]
): HubNeedsAttentionItem[] =>
  [...items].sort(
    (a, b) =>
      NEEDS_ATTENTION_RANK[a.severity] - NEEDS_ATTENTION_RANK[b.severity]
  );

/** The issues behind the rows: the section's count, and the banner's "Review N issues". */
export const attentionIssues = (items: HubNeedsAttentionItem[]): number =>
  items.reduce((sum, item) => sum + item.issueCount, 0);

/** The rows in an environment filter: a row that spans several, "prod, staging", is in each of them. */
export const attentionInEnvironment = (
  items: HubNeedsAttentionItem[],
  filter: string | null
): HubNeedsAttentionItem[] =>
  filter === null
    ? items
    : items.filter((item) =>
        item.environment
          .split(',')
          .some((environment) =>
            matchesEnvironment(environment.trim() || undefined, filter)
          )
      );
