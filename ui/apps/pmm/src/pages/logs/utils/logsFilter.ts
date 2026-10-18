import { LogFilter, SeverityClass } from 'api/logs';

export const QUICK_RANGES = ['15m', '1h', '6h', '24h', '7d'] as const;
export type QuickRange = (typeof QUICK_RANGES)[number];
export const DEFAULT_RANGE: QuickRange = '1h';

const RANGE_MS: Record<QuickRange, number> = {
  '15m': 15 * 60_000,
  '1h': 60 * 60_000,
  '6h': 6 * 60 * 60_000,
  '24h': 24 * 60 * 60_000,
  '7d': 7 * 24 * 60 * 60_000,
};

export const SEVERITY_OPTIONS = [
  { label: 'Any', value: 0 },
  { label: 'Info', value: 9 },
  { label: 'Warning', value: 13 },
  { label: 'Error', value: 17 },
  { label: 'Fatal', value: 21 },
] as const;

/** Whether any filter besides the time range is set. */
export const hasFilters = (f: LogFilter) =>
  !!(
    f.nodeNames ||
    f.sources ||
    f.presets ||
    f.minSeverity ||
    f.maxSeverity ||
    f.text ||
    f.pattern ||
    f.excludePatternHashes
  );

const isQuickRange = (v: string | null): v is QuickRange =>
  QUICK_RANGES.includes(v as QuickRange);

const absoluteRange = (
  params: URLSearchParams
): [number, number] | undefined => {
  const from = Number(params.get('from'));
  const to = Number(params.get('to'));
  if (!params.get('from') || !params.get('to') || !from || !to)
    return undefined;
  return [from, to];
};

/** The quick range in the URL, or undefined when an absolute from/to is set. */
export const rangeFromSearchParams = (
  params: URLSearchParams
): QuickRange | undefined => {
  if (absoluteRange(params)) return undefined;
  const range = params.get('range');
  return isQuickRange(range) ? range : DEFAULT_RANGE;
};

export const filterFromSearchParams = (
  params: URLSearchParams,
  now: Date
): LogFilter => {
  const abs = absoluteRange(params);
  const range = rangeFromSearchParams(params) ?? DEFAULT_RANGE;
  const [from, to] = abs ?? [now.getTime() - RANGE_MS[range], now.getTime()];
  const filter: LogFilter = {
    start: new Date(from).toISOString(),
    end: new Date(to).toISOString(),
  };
  const lists = {
    nodeNames: 'node',
    excludeNodeNames: 'exclude_node',
    sources: 'source',
    presets: 'preset',
    excludePatternHashes: 'hide',
  } as const;
  for (const [field, key] of Object.entries(lists)) {
    const values = params.getAll(key).filter(Boolean);
    if (values.length) filter[field as keyof typeof lists] = values;
  }
  const severity = Number(params.get('severity'));
  if (severity > 0 && SEVERITY_OPTIONS.some((o) => o.value === severity)) {
    filter.minSeverity = severity;
  }
  const maxSeverity = Number(params.get('severity_max'));
  if (maxSeverity > 0 && maxSeverity <= MAX_SEVERITY) {
    filter.maxSeverity = maxSeverity;
  }
  const text = params.get('q');
  if (text) filter.text = text;
  const pattern = params.get('pattern');
  if (pattern) filter.pattern = pattern;
  return filter;
};

/** URL parameters for a filter; a quick range replaces its absolute start and end. */
export const searchParamsFromFilter = (
  filter: LogFilter,
  range?: QuickRange
): URLSearchParams => {
  const params = new URLSearchParams();
  if (range) {
    params.set('range', range);
  } else {
    params.set('from', String(Date.parse(filter.start)));
    params.set('to', String(Date.parse(filter.end)));
  }
  filter.nodeNames?.forEach((v) => params.append('node', v));
  filter.excludeNodeNames?.forEach((v) => params.append('exclude_node', v));
  filter.sources?.forEach((v) => params.append('source', v));
  filter.presets?.forEach((v) => params.append('preset', v));
  if (filter.minSeverity) params.set('severity', String(filter.minSeverity));
  if (filter.maxSeverity) {
    params.set('severity_max', String(filter.maxSeverity));
  }
  if (filter.text) params.set('q', filter.text);
  if (filter.pattern) params.set('pattern', filter.pattern);
  filter.excludePatternHashes?.forEach((v) => params.append('hide', v));
  return params;
};

// The highest OpenTelemetry severity number (FATAL4).
const MAX_SEVERITY = 24;

/** PMM Server's own node, whose logs the Hide PMM Server switch excludes. */
export const PMM_SERVER_NODE = 'pmm-server';

// The severity numbers of each class, as severityClass draws them.
const CLASS_BOUNDS: Record<
  SeverityClass,
  Pick<LogFilter, 'minSeverity' | 'maxSeverity'>
> = {
  error: { minSeverity: 17 },
  warning: { minSeverity: 13, maxSeverity: 16 },
  info: { minSeverity: 9, maxSeverity: 12 },
  other: { maxSeverity: 8 },
};

/** The severity class the filter selects exactly, if any. */
export const severityClassOf = (f: LogFilter): SeverityClass | undefined =>
  (Object.keys(CLASS_BOUNDS) as SeverityClass[]).find(
    (cls) =>
      CLASS_BOUNDS[cls].minSeverity === f.minSeverity &&
      CLASS_BOUNDS[cls].maxSeverity === f.maxSeverity
  );

/** Selects only the lines of one severity class, or clears the severity when that class is already selected. */
export const toggleSeverityClass = (
  f: LogFilter,
  cls: SeverityClass
): LogFilter => {
  const rest = { ...f };
  delete rest.minSeverity;
  delete rest.maxSeverity;
  return severityClassOf(f) === cls ? rest : { ...rest, ...CLASS_BOUNDS[cls] };
};
