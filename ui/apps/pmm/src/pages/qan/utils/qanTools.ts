import { stripQanServiceId } from 'utils/qanServiceId';

export { buildNativeQanPath } from 'utils/nativeQanNav';
export {
  isQanDimensionFilterParam,
  RESERVED_FILTER_PARAM_KEYS,
} from 'pages/qan/utils/qanUrlParams';
import type { QanLabelFilter, QanLabelsMap } from 'types/qan.types';
import { asStringList } from 'pages/qan/utils/qanNormalize';
import { isQanDimensionFilterParam } from 'pages/qan/utils/qanUrlParams';

export const ALL_VARIABLE_VALUE = '$__all';
export const ALL_VARIABLE_TEXT = 'All';
export const DEFAULT_QAN_COLUMNS = ['load', 'num_queries', 'query_time'];
export const DEFAULT_PAGE_SIZE = 25;
export const DEFAULT_PAGE_NUMBER = 1;

const hasAllValueOrText = (element: string) =>
  element !== ALL_VARIABLE_VALUE && element !== ALL_VARIABLE_TEXT;

export const getLabelQueryParams = (labels: QanLabelsMap): QanLabelFilter[] =>
  Object.keys(labels)
    .filter((key) => key !== 'interval' && key !== 'by')
    .map((key) => ({
      key,
      value: asStringList(labels[key]),
    }))
    .filter((item) => item.value.filter(hasAllValueOrText).length > 0);

/** Merge `service_id` / `filter_service_id` query params into QAN label map. */
export function mergeServiceIdFromSearchParams(
  labels: QanLabelsMap,
  params: URLSearchParams
): QanLabelsMap {
  const next = { ...labels };
  const raw =
    params.get('filter_service_id') ??
    params.get('service_id') ??
    '';
  const id = stripQanServiceId(raw);
  if (!id) return next;
  const existing = next.service_id ?? [];
  if (!existing.includes(id)) {
    next.service_id = [...existing.filter(hasAllValueOrText), id];
  }
  return next;
}

export const toIsoPeriod = (unixMs: number): string =>
  new Date(unixMs).toISOString();

export const toUnixTimestamp = (iso: string): number =>
  Math.floor(new Date(iso).getTime());

/** Format ISO timestamp for `<input type="datetime-local" />`. */
export const toDatetimeLocalValue = (iso: string): string => {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
};

export function buildNativeQanShareLink(from: number, to: number): string {
  const { origin, pathname, search } = window.location;
  const params = new URLSearchParams(search.substring(1));
  params.set('from', String(from));
  params.set('to', String(to));
  return `${origin}${pathname}?${params.toString()}`;
}

export function labelsFromSearchParams(params: URLSearchParams): QanLabelsMap {
  const labels: QanLabelsMap = {};
  params.forEach((value, key) => {
    if (!isQanDimensionFilterParam(key)) return;
    const labelKey = key.slice('filter_'.length);
    labels[labelKey] = value.split(',').filter(Boolean);
  });
  return labels;
}

export function appendLabelsToSearchParams(
  params: URLSearchParams,
  labels: QanLabelsMap
): void {
  [...params.keys()]
    .filter((k) => isQanDimensionFilterParam(k))
    .forEach((k) => params.delete(k));
  Object.entries(labels).forEach(([key, values]) => {
    const filtered = asStringList(values).filter(hasAllValueOrText);
    if (filtered.length) {
      params.set(`filter_${key}`, filtered.join(','));
    }
  });
  params.delete('service_id');
  params.delete('filter_service_id');
  const sid = labels.service_id?.find(hasAllValueOrText);
  if (sid) {
    const id = stripQanServiceId(sid);
    params.set('filter_service_id', id);
    params.set('service_id', id);
  }
}
