import type { QanDatabaseType } from 'types/qan.types';

/** Map PMM management/inventory service type strings to QAN engine family. */
export function serviceTypeToQanDatabase(serviceType?: string): QanDatabaseType | undefined {
  const t = (serviceType ?? '').toLowerCase();
  if (!t) return undefined;
  if (t.includes('mongo')) return 'mongodb';
  if (t.includes('postgres')) return 'postgresql';
  if (t.includes('mysql') || t.includes('maria')) return 'mysql';
  return undefined;
}
