import { Severity } from './severity.types';

export enum AdvisorInterval {
  standard = 'ADVISOR_CHECK_INTERVAL_STANDARD',
  rare = 'ADVISOR_CHECK_INTERVAL_RARE',
  frequent = 'ADVISOR_CHECK_INTERVAL_FREQUENT',
  unspecified = 'ADVISOR_CHECK_INTERVAL_UNSPECIFIED',
}

export enum AdvisorFamily {
  unspecified = 'ADVISOR_CHECK_FAMILY_UNSPECIFIED',
  mysql = 'ADVISOR_CHECK_FAMILY_MYSQL',
  postgresql = 'ADVISOR_CHECK_FAMILY_POSTGRESQL',
  mongodb = 'ADVISOR_CHECK_FAMILY_MONGODB',
}

export interface AdvisorCheck {
  name: string;
  enabled: boolean;
  description: string;
  summary: string;
  interval: AdvisorInterval;
  family: AdvisorFamily;
}

export interface Advisor {
  name: string;
  description: string;
  summary: string;
  comment: string;
  category: string;
  checks: AdvisorCheck[];
}

export interface ListAdvisorsResponse {
  advisors: Advisor[];
}

export interface StartAdvisorChecksRequest {
  names: string[];
}

export interface ChangeAdvisorCheckParams {
  name: string;
  enable?: boolean;
  interval?: AdvisorInterval;
}

export interface ChangeAdvisorChecksRequest {
  params: ChangeAdvisorCheckParams[];
}

export enum AdvisorCheckResultStatus {
  unspecified = 'ADVISOR_CHECK_RESULT_STATUS_UNSPECIFIED',
  ok = 'ADVISOR_CHECK_RESULT_STATUS_OK',
  failed = 'ADVISOR_CHECK_RESULT_STATUS_FAILED',
  error = 'ADVISOR_CHECK_RESULT_STATUS_ERROR',
}

export enum AdvisorCheckTriggeredBy {
  unspecified = 'ADVISOR_CHECK_TRIGGERED_BY_UNSPECIFIED',
  user = 'ADVISOR_CHECK_TRIGGERED_BY_USER',
  scheduler = 'ADVISOR_CHECK_TRIGGERED_BY_SCHEDULER',
}

export interface StartAdvisorChecksResponse {
  runId: string;
}

export interface CheckResultHistoryItem {
  id: string;
  checkName: string;
  advisorName: string;
  category: string;
  interval: AdvisorInterval;
  serviceId: string;
  serviceName: string;
  serviceType: string;
  nodeId: string;
  nodeName: string;
  status: AdvisorCheckResultStatus;
  summary: string;
  description: string;
  readMoreUrl: string;
  severity: Severity;
  labels: Record<string, string>;
  checkedAt: string;
  isRead: boolean;
  runId: string;
  triggeredBy: AdvisorCheckTriggeredBy;
}

export interface ListCheckResultsHistoryParams {
  pageSize?: number;
  pageIndex?: number;
  serviceId?: string;
  serviceName?: string;
  nodeName?: string;
  category?: string;
  checkName?: string;
  status?: AdvisorCheckResultStatus;
  severity?: Severity;
  isRead?: boolean;
  runId?: string;
  triggeredBy?: AdvisorCheckTriggeredBy;
  from?: string;
  to?: string;
}

export interface MarkCheckResultsReadRequest {
  ids: string[];
  isRead: boolean;
}

export interface ListCheckResultsFilterValuesResponse {
  serviceNames: string[];
  nodeNames: string[];
}

export interface AdvisorCheckRow {
  checkName: string;
  summary: string;
  description: string;
  advisorName: string;
  category: string;
  family: AdvisorFamily;
  interval: AdvisorInterval;
  enabled: boolean;
}
