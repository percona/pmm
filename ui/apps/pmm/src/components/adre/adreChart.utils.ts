/** A chart the Assistant asked for: the query, which datasource runs it and over which window. */
export interface ChartSpec {
  ds: string;
  title?: string;
  from: string;
  to: string;
  unit?: string;
  query: string;
}

const ATTR_RE = /(\w+)=(?:"([^"]*)"|(\S+))/g;

/** The spec of a ```chart block, from its first-line attributes and its body; undefined without a query. */
export const chartSpecFromBlock = (
  meta: string | undefined,
  code: string
): ChartSpec | undefined => {
  const query = code.trim();
  if (!query) return undefined;
  const attrs: Record<string, string> = {};
  for (const m of (meta ?? '').matchAll(ATTR_RE)) {
    attrs[m[1]] = m[2] ?? m[3];
  }
  return {
    ds: attrs.ds || 'Metrics',
    ...(attrs.title && { title: attrs.title }),
    from: attrs.from || 'now-1h',
    to: attrs.to || 'now',
    ...(attrs.unit && { unit: attrs.unit }),
    query,
  };
};

const UNIT_MS: Record<string, number> = {
  s: 1000,
  m: 60_000,
  h: 3600_000,
  d: 86_400_000,
  w: 604_800_000,
};

const timeMs = (t: string, now: number) => {
  const rel = /^now(?:-(\d+)([smhdw]))?$/.exec(t.trim());
  if (rel) return rel[1] ? now - Number(rel[1]) * UNIT_MS[rel[2]] : now;
  const ms = Date.parse(t);
  return Number.isNaN(ms) ? undefined : ms;
};

/** The window as epoch milliseconds: RFC3339 or now-<n><s|m|h|d|w>; undefined when it does not parse or is empty. */
export const rangeMs = (from: string, to: string, now = Date.now()) => {
  const f = timeMs(from, now);
  const t = timeMs(to, now);
  if (f === undefined || t === undefined || f >= t) return undefined;
  return { from: f, to: t };
};

export interface DataFrameJSON {
  schema: {
    fields: {
      name: string;
      type: string;
      labels?: Record<string, string>;
      config?: { displayNameFromDS?: string };
    }[];
  };
  data: { values: unknown[][] };
}

export interface ChartSeries {
  label: string;
  data: (number | null)[];
}

const seriesLabel = (field: DataFrameJSON['schema']['fields'][number]) => {
  const labels = Object.entries(field.labels ?? {});
  if (labels.length === 1) return labels[0][1];
  if (labels.length > 1) return labels.map(([k, v]) => `${k}=${v}`).join(', ');
  return field.config?.displayNameFromDS ?? field.name;
};

const peak = (s: ChartSeries) =>
  s.data.reduce<number>((m, v) => (v !== null && v > m ? v : m), -Infinity);

/** Grafana data frames as series on one shared time axis, the largest first, at most maxSeries of them. */
export const framesToSeries = (frames: DataFrameJSON[], maxSeries: number) => {
  const columns: { label: string; points: Map<number, number> }[] = [];
  const allTimes = new Set<number>();
  for (const frame of frames) {
    const fields = frame.schema.fields;
    const timeIdx = fields.findIndex((f) => f.type === 'time');
    if (timeIdx < 0) continue;
    const times = frame.data.values[timeIdx] as number[];
    times.forEach((t) => allTimes.add(t));
    fields.forEach((field, i) => {
      if (field.type !== 'number') return;
      const values = frame.data.values[i] as (number | null)[];
      const points = new Map<number, number>();
      times.forEach((t, j) => {
        const v = values[j];
        if (typeof v === 'number' && Number.isFinite(v)) points.set(t, v);
      });
      columns.push({ label: seriesLabel(field), points });
    });
  }
  const times = [...allTimes].sort((a, b) => a - b);
  const series = columns
    .map(({ label, points }) => ({
      label,
      data: times.map((t) => points.get(t) ?? null),
    }))
    .sort((a, b) => peak(b) - peak(a));
  return {
    times,
    series: series.slice(0, maxSeries),
    hidden: Math.max(0, series.length - maxSeries),
  };
};

const compact = new Intl.NumberFormat('en', {
  notation: 'compact',
  maximumFractionDigits: 2,
});
const BYTE_UNITS = ['B', 'KiB', 'MiB', 'GiB', 'TiB', 'PiB'];

/** A value in the chart's unit: percent (0–100), percentunit (0–1), bytes, s, ms, ops, or plain. */
export const formatValue = (v: number | null, unit?: string): string => {
  if (v === null) return '';
  switch (unit) {
    case 'percent':
      return `${v.toFixed(1)}%`;
    case 'percentunit':
      return `${(v * 100).toFixed(1)}%`;
    case 'bytes': {
      let i = 0;
      let n = Math.abs(v);
      while (n >= 1024 && i < BYTE_UNITS.length - 1) {
        n /= 1024;
        i += 1;
      }
      return `${(Math.sign(v) * n).toFixed(n < 10 && i > 0 ? 1 : 0)} ${BYTE_UNITS[i]}`;
    }
    case 's':
      return Math.abs(v) < 1
        ? `${(v * 1000).toFixed(0)} ms`
        : `${compact.format(v)} s`;
    case 'ms':
      return Math.abs(v) >= 1000
        ? `${compact.format(v / 1000)} s`
        : `${v.toFixed(0)} ms`;
    case 'ops':
      return `${compact.format(v)}/s`;
    default:
      return compact.format(v);
  }
};
