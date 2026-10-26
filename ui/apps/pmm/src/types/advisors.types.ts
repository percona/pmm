import type { Severity } from './severity.types';

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
  technology: AdvisorTechnology;
  category: string;
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
  technology: AdvisorTechnology;
  queries: AdvisorCheckQuery[];
  script: string;
}

export interface ListAdvisorChecksResponse {
  checks: AdvisorCheck[];
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
  // the check is named by the request path
  check: Omit<AdvisorCheckInput, 'name'>;
}

export interface UpdateAdvisorCheckResponse {
  check: AdvisorCheck;
}

export interface StartAdvisorChecksRequest {
  // when set, only these checks run
  names?: string[];
  // when set, the checks run only against these services
  serviceIds?: string[];
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
  pending = 'ADVISOR_CHECK_RESULT_STATUS_PENDING',
  notRun = 'ADVISOR_CHECK_RESULT_STATUS_NOT_RUN',
}

export enum AdvisorCheckTriggeredBy {
  unspecified = 'ADVISOR_CHECK_TRIGGERED_BY_UNSPECIFIED',
  user = 'ADVISOR_CHECK_TRIGGERED_BY_USER',
  scheduler = 'ADVISOR_CHECK_TRIGGERED_BY_SCHEDULER',
}

export interface SendTestAdvisorNotificationRequest {
  emailAddresses: string[];
  severities: Severity[];
}

export interface StartAdvisorChecksResponse {
  runId: string;
}

export interface Insight {
  id: string;
  checkName: string;
  category: string;
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
  // unset while the check is pending and when it did not run
  checkedAt?: string | null;
  runId: string;
  triggeredBy: AdvisorCheckTriggeredBy;
  outcome: string;
  environment: string;
  cluster: string;
  replicationSet: string;
  region: string;
  az: string;
}

export interface SeverityCount {
  severity: Severity;
  count: number;
}

export enum AdvisorRunStatus {
  unspecified = 'ADVISOR_RUN_STATUS_UNSPECIFIED',
  queued = 'ADVISOR_RUN_STATUS_QUEUED',
  running = 'ADVISOR_RUN_STATUS_RUNNING',
  completed = 'ADVISOR_RUN_STATUS_COMPLETED',
  interrupted = 'ADVISOR_RUN_STATUS_INTERRUPTED',
  aborted = 'ADVISOR_RUN_STATUS_ABORTED',
}

// A single execution of Advisor checks. While it runs, its totals cover the
// checks executed so far.
export interface AdvisorRun {
  id: string;
  triggeredBy: AdvisorCheckTriggeredBy;
  status: AdvisorRunStatus;
  startedAt: string;
  // unset while the run is queued or running
  finishedAt?: string | null;
  // what the run planned to cover; zero until it starts
  plannedChecksCount: number;
  plannedServicesCount: number;
  // checks and services with a passed or failed outcome
  checksCount: number;
  servicesCount: number;
  findingsCount: number;
  errorsCount: number;
  severityCounts: SeverityCount[];
  // the scope the run was narrowed to; empty means all
  checkNames: string[];
  serviceIds: string[];
  // the schedule the run ran for or was started from; empty for other runs
  // and once the schedule is deleted
  scheduleId?: string;
  // the schedule's name when the run started, kept after a rename or delete
  scheduleName?: string;
  // when the schedule fell due; unset for runs a user started
  scheduledFor?: string | null;
  // set only when a scheduled run started over a minute late behind this run
  waitedForRun?: AdvisorRunReference | null;
}

export interface AdvisorRunReference {
  runId: string;
  triggeredBy: AdvisorCheckTriggeredBy;
  scheduleName?: string;
  startedAt: string;
}

export interface ListRunsParams {
  pageSize?: number;
  pageIndex?: number;
  runId?: string;
  triggeredBy?: AdvisorCheckTriggeredBy;
  scheduleId?: string;
  from?: string;
  to?: string;
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
  runId?: string;
  triggeredBy?: AdvisorCheckTriggeredBy;
  from?: string;
  to?: string;
}

// the filters the Insights page sets, which EmailInsights takes too
export type InsightsFilterParams = Pick<
  ListInsightsParams,
  | 'runId'
  | 'serviceName'
  | 'nodeName'
  | 'category'
  | 'checkName'
  | 'severity'
  | 'status'
>;

export interface EmailInsightsRequest extends InsightsFilterParams {
  emailAddresses: string[];
}

export interface EmailInsightsResponse {
  insightsCount: number;
}

export interface ListInsightsFilterValuesResponse {
  serviceNames: string[];
  nodeNames: string[];
}

export interface AdvisorCheckRow {
  checkName: string;
  summary: string;
  description: string;
  category: string;
  technology: AdvisorTechnology;
  enabled: boolean;
  userDefined: boolean;
  disabledServiceIds: string[];
}

export enum AdvisorScheduleEmailMode {
  unspecified = 'ADVISOR_SCHEDULE_EMAIL_MODE_UNSPECIFIED',
  settings = 'ADVISOR_SCHEDULE_EMAIL_MODE_SETTINGS',
  custom = 'ADVISOR_SCHEDULE_EMAIL_MODE_CUSTOM',
  off = 'ADVISOR_SCHEDULE_EMAIL_MODE_OFF',
}

// Runs Advisor checks on the given days at a time of day, both in UTC.
// false booleans and empty lists are omitted from the JSON
export interface AdvisorSchedule {
  scheduleId: string;
  name: string;
  enabled?: boolean;
  // 0 is Sunday, as Date.getUTCDay()
  daysOfWeek: number[];
  // HH:MM, UTC
  timeOfDay: string;
  // empty means all, including checks and services added later
  checkNames?: string[];
  serviceIds?: string[];
  emailMode: AdvisorScheduleEmailMode;
  // used only by the custom email mode
  emailAddresses?: string[];
  // empty means the default severities from Settings
  emailSeverities?: Severity[];
  // unset while the schedule is disabled
  nextRunAt?: string | null;
  lastRun?: AdvisorRun | null;
  createdAt: string;
  updatedAt: string;
}

export interface ListAdvisorSchedulesResponse {
  schedules?: AdvisorSchedule[];
}

// the full schedule: an update replaces it, so toggling sends every field
export interface AdvisorScheduleInput {
  name: string;
  enabled: boolean;
  daysOfWeek: number[];
  timeOfDay: string;
  checkNames: string[];
  serviceIds: string[];
  emailMode: AdvisorScheduleEmailMode;
  emailAddresses: string[];
  emailSeverities: Severity[];
}

export interface AdvisorScheduleResponse {
  schedule: AdvisorSchedule;
}

export interface RunAdvisorScheduleResponse {
  runId: string;
}

// a MySQL, PostgreSQL or MongoDB service Advisor checks can run against
export interface AdvisorTargetService {
  serviceId: string;
  serviceName: string;
  technology: AdvisorTechnology;
  nodeName?: string;
  environment?: string;
  cluster?: string;
}

export interface ListAdvisorServicesResponse {
  services?: AdvisorTargetService[];
}

// the email defaults from Settings that schedules fall back to
export interface AdvisorNotificationSettings {
  enabled?: boolean;
  emailAddresses?: string[];
  severities?: Severity[];
}
