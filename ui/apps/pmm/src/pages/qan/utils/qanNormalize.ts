import type { QanFilterLabelValue, QanGetMetricNamesResponse } from 'types/qan.types';
import { DEFAULT_QAN_COLUMNS } from './qanTools';

/** qan-api2 returns metric names as a map (key → label), not an array. */
export function metricNamesFromResponse(
  response: QanGetMetricNamesResponse | undefined
): string[] {
  const raw = response?.data;
  if (!raw) return [];
  if (Array.isArray(raw)) {
    return raw
      .map((item) => (typeof item === 'string' ? item : item?.name))
      .filter((name): name is string => typeof name === 'string' && name.length > 0);
  }
  if (typeof raw === 'object') {
    return Object.keys(raw);
  }
  return [];
}

export function parseQanColumns(raw: string | null): string[] {
  if (!raw) return DEFAULT_QAN_COLUMNS;
  try {
    const parsed: unknown = JSON.parse(raw);
    if (Array.isArray(parsed) && parsed.every((c) => typeof c === 'string')) {
      return parsed.length ? parsed : DEFAULT_QAN_COLUMNS;
    }
  } catch {
    /* use default */
  }
  return DEFAULT_QAN_COLUMNS;
}

export function asLabelValueList(value: unknown): QanFilterLabelValue[] {
  return Array.isArray(value) ? value : [];
}

export function asStringList(value: unknown): string[] {
  if (Array.isArray(value)) {
    return value.filter((v): v is string => typeof v === 'string');
  }
  if (typeof value === 'string') {
    return value.split(',').filter(Boolean);
  }
  return [];
}
