import { get } from './api';
import type {
  CheckResultSummaryPayload,
  FailedCheckSummary,
  FailedServicesResponse,
} from '../types/advisors.types';

const toCount = (value = '0') => parseInt(value, 10);

const toFailedCheckSummary = ({
  service_name,
  service_id,
  emergency_count,
  alert_count,
  critical_count,
  error_count,
  warning_count,
  notice_count,
  info_count,
  debug_count,
}: CheckResultSummaryPayload): FailedCheckSummary => ({
  serviceName: service_name,
  serviceId: service_id,
  counts: {
    emergency: toCount(emergency_count),
    alert: toCount(alert_count),
    critical: toCount(critical_count),
    error: toCount(error_count),
    warning: toCount(warning_count),
    notice: toCount(notice_count),
    info: toCount(info_count),
    debug: toCount(debug_count),
  },
});

export const getFailedServices = async (): Promise<FailedCheckSummary[]> => {
  const { result = [] } = await get<FailedServicesResponse>(
    'advisors/failedServices'
  );

  return result
    .map(toFailedCheckSummary)
    .sort((a, b) => a.serviceName.localeCompare(b.serviceName));
};
