/** Local time of day with milliseconds, e.g. 14:39:58.412. */
export const timeOfDay = (iso: string) => {
  const d = new Date(iso);
  const time = d.toLocaleTimeString(undefined, {
    hour12: false,
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
  return `${time}.${String(d.getMilliseconds()).padStart(3, '0')}`;
};

/** Local calendar day, e.g. Saturday, October 17, 2026. */
export const dayLabel = (iso: string) =>
  new Date(iso).toLocaleDateString(undefined, {
    weekday: 'long',
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });

/** Local day and minute, e.g. Oct 17, 14:39. */
export const shortDateTime = (iso: string) =>
  new Date(iso).toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });

const DURATION_UNITS: [number, string][] = [
  [86_400_000, 'day'],
  [3_600_000, 'hour'],
  [60_000, 'minute'],
];

/** A duration in its largest whole unit, e.g. 6 hours. */
export const formatDuration = (ms: number) => {
  const [size, unit] = DURATION_UNITS.find(([size]) => ms >= size) ?? [
    1000,
    'second',
  ];
  const n = Math.round(ms / size);
  return `${n} ${unit}${n === 1 ? '' : 's'}`;
};
