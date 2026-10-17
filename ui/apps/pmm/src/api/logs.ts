import axios from 'axios';
import { api } from './api';

export interface LogFilter {
  start: string;
  end: string;
  nodeNames?: string[];
  excludeNodeNames?: string[];
  sources?: string[];
  presets?: string[];
  minSeverity?: number;
  maxSeverity?: number;
  text?: string;
  pattern?: string;
}

export interface LogLine {
  time: string;
  nodeName: string;
  source: string;
  preset: string;
  fileName: string;
  severityNumber: number;
  severityText: string;
  body: string;
  bodyTruncated: boolean;
  attributes: Record<string, string>;
}

export interface VolumeBucket {
  start: string;
  error: number;
  warning: number;
  info: number;
  other: number;
}

export interface LogPattern {
  pattern: string;
  count: number;
  maxSeverityNumber: number;
  firstSeen: string;
  lastSeen: string;
  sample: string;
  isNew: boolean;
}

export interface LogFacet {
  value: string;
  count: number;
}

export interface LogFacets {
  nodeNames: LogFacet[];
  sources: LogFacet[];
  presets: LogFacet[];
}

export type SeverityClass = 'error' | 'warning' | 'info' | 'other';

/** OpenTelemetry severity number to the class the page colours and filters by. */
export const severityClass = (severityNumber: number): SeverityClass => {
  if (severityNumber >= 17) return 'error';
  if (severityNumber >= 13) return 'warning';
  if (severityNumber >= 9) return 'info';
  return 'other';
};

const SEVERITY_LABELS = ['TRACE', 'DEBUG', 'INFO', 'WARN', 'ERROR', 'FATAL'];

/** The OpenTelemetry name of a severity number's range, the same for every source; empty when unset. */
export const severityLabel = (severityNumber: number): string =>
  severityNumber > 0
    ? SEVERITY_LABELS[Math.min(5, Math.floor((severityNumber - 1) / 4))]
    : '';

// Wire types: snake_case, int64 as strings, zero values may be omitted.
interface RawLogLine {
  time: string;
  node_name?: string;
  source?: string;
  preset?: string;
  file_name?: string;
  severity_number?: number;
  severity_text?: string;
  body?: string;
  body_truncated?: boolean;
  attributes?: Record<string, string>;
}

interface RawFacet {
  value?: string;
  count?: string;
}

// Responses skip the shared case converter, which would also rename the keys of `attributes` (log.file.name).
const config = {
  disableNotifications: true,
  transformResponse: axios.defaults.transformResponse,
};

const toRawFilter = (f: LogFilter) => ({
  start: f.start,
  end: f.end,
  node_names: f.nodeNames,
  sources: f.sources,
  presets: f.presets,
  min_severity: f.minSeverity,
  max_severity: f.maxSeverity,
  exclude_node_names: f.excludeNodeNames,
  text: f.text,
  pattern: f.pattern,
});

const toLogLine = (l: RawLogLine): LogLine => ({
  time: l.time,
  nodeName: l.node_name ?? '',
  source: l.source ?? '',
  preset: l.preset ?? '',
  fileName: l.file_name ?? '',
  severityNumber: l.severity_number ?? 0,
  severityText: l.severity_text ?? '',
  body: l.body ?? '',
  bodyTruncated: !!l.body_truncated,
  attributes: l.attributes ?? {},
});

const toFacets = (facets?: RawFacet[]): LogFacet[] =>
  (facets ?? []).map((f) => ({
    value: f.value ?? '',
    count: Number(f.count ?? 0),
  }));

export const searchLogs = async (
  filter: LogFilter,
  cursor?: string,
  limit?: number
): Promise<{ lines: LogLine[]; nextCursor?: string }> => {
  const res = await api.post<{ lines?: RawLogLine[]; next_cursor?: string }>(
    '/logs:search',
    { filter: toRawFilter(filter), cursor, limit },
    config
  );
  return {
    lines: (res.data.lines ?? []).map(toLogLine),
    nextCursor: res.data.next_cursor || undefined,
  };
};

export const getLogVolume = async (
  filter: LogFilter
): Promise<{ buckets: VolumeBucket[]; step: string }> => {
  const res = await api.post<{
    buckets?: {
      start: string;
      error?: string;
      warning?: string;
      info?: string;
      other?: string;
    }[];
    step?: string;
  }>('/logs:getVolume', { filter: toRawFilter(filter) }, config);
  return {
    buckets: (res.data.buckets ?? []).map((b) => ({
      start: b.start,
      error: Number(b.error ?? 0),
      warning: Number(b.warning ?? 0),
      info: Number(b.info ?? 0),
      other: Number(b.other ?? 0),
    })),
    step: res.data.step ?? '0s',
  };
};

export const getLogPatterns = async (
  filter: LogFilter
): Promise<LogPattern[]> => {
  const res = await api.post<{
    patterns?: {
      pattern?: string;
      count?: string;
      max_severity_number?: number;
      first_seen: string;
      last_seen: string;
      sample?: string;
      is_new?: boolean;
    }[];
  }>('/logs:getPatterns', { filter: toRawFilter(filter) }, config);
  return (res.data.patterns ?? []).map((p) => ({
    pattern: p.pattern ?? '',
    count: Number(p.count ?? 0),
    maxSeverityNumber: p.max_severity_number ?? 0,
    firstSeen: p.first_seen,
    lastSeen: p.last_seen,
    sample: p.sample ?? '',
    isNew: !!p.is_new,
  }));
};

export const getLogContext = async (
  time: string,
  nodeName: string,
  fileName: string,
  source?: string,
  lines?: number
): Promise<{ lines: LogLine[]; anchorIndex: number }> => {
  const res = await api.post<{ lines?: RawLogLine[]; anchor_index?: number }>(
    '/logs:getContext',
    { time, node_name: nodeName, file_name: fileName, source, lines },
    config
  );
  return {
    lines: (res.data.lines ?? []).map(toLogLine),
    anchorIndex: res.data.anchor_index ?? 0,
  };
};

export const getLogFacets = async (filter: LogFilter): Promise<LogFacets> => {
  const res = await api.post<{
    node_names?: RawFacet[];
    sources?: RawFacet[];
    presets?: RawFacet[];
  }>('/logs:getFacets', { filter: toRawFilter(filter) }, config);
  return {
    nodeNames: toFacets(res.data.node_names),
    sources: toFacets(res.data.sources),
    presets: toFacets(res.data.presets),
  };
};
