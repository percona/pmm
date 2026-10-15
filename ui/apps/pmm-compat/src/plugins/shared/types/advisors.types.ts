export interface CheckResultSummaryPayload {
  service_name: string;
  service_id: string;
  emergency_count?: string;
  alert_count?: string;
  critical_count?: string;
  error_count?: string;
  warning_count?: string;
  notice_count?: string;
  info_count?: string;
  debug_count?: string;
}

export interface FailedServicesResponse {
  result?: CheckResultSummaryPayload[];
}

export interface FailedChecksCounts {
  emergency: number;
  alert: number;
  critical: number;
  error: number;
  warning: number;
  notice: number;
  info: number;
  debug: number;
}

export interface FailedCheckSummary {
  serviceName: string;
  serviceId: string;
  counts: FailedChecksCounts;
}
