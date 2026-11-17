import type { QanDatabaseType } from 'types/qan.types';

export function inferDatabaseType(database?: string): QanDatabaseType {
  const d = (database ?? '').toLowerCase();
  if (d.includes('mongo')) return 'mongodb';
  if (d.includes('postgres') || d.includes('pg')) return 'postgresql';
  if (d.includes('mysql') || d.includes('maria')) return 'mysql';
  return 'unknown';
}

/** @deprecated Prefer resolveServiceUuid / useQanServiceId for API calls. */
export function serviceIdFromLabels(labels: Record<string, string[]>): string {
  const ids = labels.service_id ?? [];
  const first = ids.find((n) => n && n !== '$__all');
  return first ?? '';
}
