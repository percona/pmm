import type { HubAttentionItem, HubSeverity } from './hub.types';

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

/** Every error stays visible, then further rows in order until `max` in total; the rest are hidden. */
export const capAttention = (
  items: HubAttentionItem[],
  max: number
): { visible: HubAttentionItem[]; hidden: HubAttentionItem[] } => {
  const errors = items.filter((item) => item.severity === 'error').length;
  let room = Math.max(0, max - errors);
  const visible: HubAttentionItem[] = [];
  const hidden: HubAttentionItem[] = [];

  for (const item of items) {
    if (item.severity === 'error') {
      visible.push(item);
    } else if (room > 0) {
      visible.push(item);
      room -= 1;
    } else {
      hidden.push(item);
    }
  }

  return { visible, hidden };
};
