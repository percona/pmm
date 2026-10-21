/** `filter_*` query params that are not QAN dimension labels. */
export const RESERVED_FILTER_PARAM_KEYS = new Set(['by', 'service_id']);

export function isQanDimensionFilterParam(paramKey: string): boolean {
  if (!paramKey.startsWith('filter_')) return false;
  const labelKey = paramKey.slice('filter_'.length);
  return !RESERVED_FILTER_PARAM_KEYS.has(labelKey);
}
