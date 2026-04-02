/** Matches QAN / Grafana `SERVICE_ID_PREFIX` so API and deep links use bare service UUIDs. */
export const QAN_SERVICE_ID_PREFIX = '/service_id/';

export function stripQanServiceId(serviceId: string): string {
  const t = String(serviceId ?? '').trim();
  if (t.startsWith(QAN_SERVICE_ID_PREFIX)) {
    return t.slice(QAN_SERVICE_ID_PREFIX.length);
  }
  return t;
}
