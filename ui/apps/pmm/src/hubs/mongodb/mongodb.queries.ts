// The module, not the package index, which would load the whole Operations UI.
import { formatCompactDuration } from '@pmm/plugins-om/src/format';
import type { HubQueries, HubQueryRow } from 'hubs/template/hub.types';
import { queryQan } from './mongodb.doors';
import { Messages } from './mongodb.messages';
import { MONGODB_HUB_THRESHOLDS as T } from './mongodb.thresholds';
import type {
  MongoServiceInput,
  MongoSnapshot,
  QanRowInput,
  RtaQueryInput,
} from './mongodb.types';

export interface SlowerQuery {
  query: QanRowInput;
  usualSeconds: number;
  ratio: number;
}

export interface RunningQuery {
  query: RtaQueryInput;
  service?: MongoServiceInput;
}

export interface QueriesDerivation {
  queries: HubQueries;
  slower: SlowerQuery[];
  running: RunningQuery[];
}

const compact = new Intl.NumberFormat('en', {
  notation: 'compact',
  maximumFractionDigits: 1,
});

export const formatCalls = (calls: number) =>
  compact.format(calls).toLowerCase();

export const formatSeconds = (seconds: number) =>
  seconds < 10 ? `${seconds.toFixed(2)}s` : `${Math.round(seconds)}s`;

/** Whole multiples read as "6×", smaller ones keep a decimal, "3.5×". */
export const formatRatio = (ratio: number) =>
  ratio >= 10 || Number.isInteger(Math.round(ratio * 10) / 10)
    ? String(Math.round(ratio))
    : (Math.round(ratio * 10) / 10).toFixed(1);

/** Queries at least 3× slower than in the day before, with enough calls and time to matter; no baseline, no row. */
export const findSlowerQueries = (
  current: QanRowInput[],
  baseline: QanRowInput[]
): SlowerQuery[] => {
  const usual = new Map(baseline.map((row) => [row.queryId, row]));

  return current
    .flatMap((query): SlowerQuery[] => {
      const before = usual.get(query.queryId);

      if (!before || before.avgQueryTime <= 0) {
        return [];
      }
      const ratio = query.avgQueryTime / before.avgQueryTime;

      return ratio >= T.querySlowerRatio &&
        query.numQueries >= T.querySlowerMinCalls &&
        query.avgQueryTime >= T.querySlowerMinAvgSeconds
        ? [{ query, usualSeconds: before.avgQueryTime, ratio }]
        : [];
    })
    .sort((a, b) => b.ratio - a.ratio);
};

/** Queries running for a minute or more on a service that has a real-time session. */
export const findRunningQueries = (
  realtime: NonNullable<MongoSnapshot['realtime']>,
  services: MongoServiceInput[]
): RunningQuery[] => {
  const byId = new Map(services.map((service) => [service.serviceId, service]));
  const sessions = new Set(realtime.sessionServiceIds);

  return realtime.queries
    .filter(
      (query) =>
        sessions.has(query.serviceId) &&
        byId.has(query.serviceId) &&
        query.durationMs >= T.queryRunningLongSeconds * 1000
    )
    .map((query) => ({ query, service: byId.get(query.serviceId) }))
    .sort((a, b) => b.query.durationMs - a.query.durationMs);
};

export const runningAge = (query: RtaQueryInput) =>
  formatCompactDuration(query.durationMs / 1000) || '0s';

export const slowerRow = (
  { query, usualSeconds, ratio }: SlowerQuery,
  filter: string | null
): HubQueryRow => ({
  id: `slower:${query.queryId}`,
  text: query.fingerprint,
  headline: Messages.queries.slower(formatRatio(ratio)),
  detail: Messages.queries.slowerDetail(
    formatSeconds(query.avgQueryTime),
    formatSeconds(usualSeconds),
    formatCalls(query.numQueries)
  ),
  severity: 'warning',
  door: {
    label: Messages.doors.openQuery,
    to: queryQan(filter, query.queryId),
  },
});

export const deriveQueries = (
  snapshot: Pick<MongoSnapshot, 'qan' | 'realtime'>,
  services: MongoServiceInput[],
  filter: string | null
): QueriesDerivation => {
  const slower = snapshot.qan
    ? findSlowerQueries(snapshot.qan.current, snapshot.qan.baseline)
    : [];
  const running = snapshot.realtime
    ? findRunningQueries(snapshot.realtime, services)
    : [];

  return {
    queries: {
      slower: slower
        .slice(0, T.queryRowsMax)
        .map((item) => slowerRow(item, filter)),
    },
    slower,
    running,
  };
};
