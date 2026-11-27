import { HealthStatus, ServiceMapOptions } from '../types';

/** Edge/node health from error rate, TCP failures, and request rate (shared by transform + pod aggregation). */
export function computeHealth(
  errPct: number,
  tcpFailed: number,
  totalRps: number,
  opts: ServiceMapOptions
): HealthStatus {
  if (totalRps <= 0 && tcpFailed <= 0) {
    return 'unknown';
  }
  if (errPct >= opts.errorRedThreshold || tcpFailed > 0) {
    return 'red';
  }
  if (errPct >= opts.errorAmberThreshold) {
    return 'amber';
  }
  return 'green';
}
