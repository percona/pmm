import { Severity } from './severity.types';

export enum AdvisorInterval {
  standard = 'ADVISOR_CHECK_INTERVAL_STANDARD',
  rare = 'ADVISOR_CHECK_INTERVAL_RARE',
  frequent = 'ADVISOR_CHECK_INTERVAL_FREQUENT',
  unspecified = 'ADVISOR_CHECK_INTERVAL_UNSPECIFIED',
}

export enum AdvisorTechnology {
  unspecified = 'ADVISOR_CHECK_TECHNOLOGY_UNSPECIFIED',
  mysql = 'ADVISOR_CHECK_TECHNOLOGY_MYSQL',
  postgresql = 'ADVISOR_CHECK_TECHNOLOGY_POSTGRESQL',
  mongodb = 'ADVISOR_CHECK_TECHNOLOGY_MONGODB',
}

export interface AdvisorCheckQuery {
  type: string;
  query: string;
  parameters?: Record<string, string>;
}

export interface AdvisorCheck {
  name: string;
  enabled: boolean;
  description: string;
  summary: string;
  interval: AdvisorInterval;
  technology: AdvisorTechnology;
  category: string;
  subcategory: string;
  userDefined: boolean;
  // IDs of services for which this check is disabled
  disabledServiceIds?: string[];
  // populated by get/create/update, empty in list responses
  queries?: AdvisorCheckQuery[];
  script?: string;
}

// AdvisorCheckInput is the authorable payload sent when creating or updating a check.
export interface AdvisorCheckInput {
  name: string;
  summary: string;
  description: string;
  category: string;
  subcategory: string;
  technology: AdvisorTechnology;
  interval: AdvisorInterval;
  queries: AdvisorCheckQuery[];
  script: string;
}

export interface Advisor {
  category: string;
  subcategory: string;
  checks: AdvisorCheck[];
}

export interface ListAdvisorsResponse {
  advisors: Advisor[];
}

export interface GetAdvisorCheckResponse {
  check: AdvisorCheck;
}

export interface CreateAdvisorCheckRequest {
  check: AdvisorCheckInput;
}

export interface CreateAdvisorCheckResponse {
  check: AdvisorCheck;
}

export interface UpdateAdvisorCheckRequest {
  check: AdvisorCheckInput;
}

export interface UpdateAdvisorCheckResponse {
  check: AdvisorCheck;
}

export interface StartAdvisorChecksRequest {
  names: string[];
}

export interface TestAdvisorCheckRequest {
  check: AdvisorCheckInput;
  serviceId: string;
}

// a single finding produced by a dry-run check execution
export interface TestAdvisorCheckResult {
  summary: string;
  checkName: string;
  description: string;
  readMoreUrl: string;
  severity: Severity;
  labels: Record<string, string>;
  serviceName: string;
  serviceId: string;
}

export interface TestAdvisorCheckResponse {
  results?: TestAdvisorCheckResult[];
  // output of the script's print() calls, for debugging
  scriptOutput?: string;
}

// a service an advisor check can be tested against
export interface AdvisorCheckTestTarget {
  serviceId: string;
  serviceName: string;
}

export interface ListAdvisorCheckTestTargetsResponse {
  targets?: AdvisorCheckTestTarget[];
}

export interface ChangeAdvisorCheckParams {
  name: string;
  enable?: boolean;
  interval?: AdvisorInterval;
  // when set, enable/disable applies only to these services
  serviceIds?: string[];
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
  batchId: string;
}

export interface Insight {
  id: string;
  checkName: string;
  subcategory: string;
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
  batchId: string;
  triggeredBy: AdvisorCheckTriggeredBy;
  outcome: string;
  environment: string;
  cluster: string;
  replicationSet: string;
}

export interface ListInsightsParams {
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
  batchId?: string;
  triggeredBy?: AdvisorCheckTriggeredBy;
  from?: string;
  to?: string;
}

export interface InsightsFilters {
  serviceName?: string;
  nodeName?: string;
  category?: string;
  severity?: Severity;
  status?: AdvisorCheckResultStatus;
  isRead?: boolean;
  batchId?: string;
}

export interface MarkInsightsReadRequest {
  // record IDs to update; takes precedence over filters
  ids?: string[];
  isRead: boolean;
  // when set and ids is empty, updates all records matching the filters
  filters?: InsightsFilters;
}

export interface ListInsightsFilterValuesResponse {
  serviceNames: string[];
  nodeNames: string[];
}

export interface AdvisorCheckRow {
  checkName: string;
  summary: string;
  description: string;
  subcategory: string;
  category: string;
  technology: AdvisorTechnology;
  interval: AdvisorInterval;
  enabled: boolean;
  userDefined: boolean;
  disabledServiceIds: string[];
}
