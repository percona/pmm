export enum ParamType {
  UNSPECIFIED = 'PARAM_TYPE_UNSPECIFIED',
  BOOL = 'PARAM_TYPE_BOOL',
  FLOAT = 'PARAM_TYPE_FLOAT',
  STRING = 'PARAM_TYPE_STRING',
}

export enum ParamUnit {
  UNSPECIFIED = 'PARAM_UNIT_UNSPECIFIED',
  PERCENTAGE = 'PARAM_UNIT_PERCENTAGE',
  SECONDS = 'PARAM_UNIT_SECONDS',
}

export enum TemplateSource {
  UNSPECIFIED = 'TEMPLATE_SOURCE_UNSPECIFIED',
  BUILT_IN = 'TEMPLATE_SOURCE_BUILT_IN',
  SAAS = 'TEMPLATE_SOURCE_SAAS',
  USER_FILE = 'TEMPLATE_SOURCE_USER_FILE',
  USER_API = 'TEMPLATE_SOURCE_USER_API',
}

export enum FilterType {
  UNSPECIFIED = 'FILTER_TYPE_UNSPECIFIED',
  MATCH = 'FILTER_TYPE_MATCH',
  MISMATCH = 'FILTER_TYPE_MISMATCH',
}

export enum Severity {
  UNSPECIFIED = 'SEVERITY_UNSPECIFIED',
  EMERGENCY = 'SEVERITY_EMERGENCY',
  ALERT = 'SEVERITY_ALERT',
  CRITICAL = 'SEVERITY_CRITICAL',
  ERROR = 'SEVERITY_ERROR',
  WARNING = 'SEVERITY_WARNING',
  NOTICE = 'SEVERITY_NOTICE',
  INFO = 'SEVERITY_INFO',
  DEBUG = 'SEVERITY_DEBUG',
}

export enum TemplateCategory {
  UNSPECIFIED = 'TEMPLATE_CATEGORY_UNSPECIFIED',
  PMM = 'TEMPLATE_CATEGORY_PMM',
  MONGODB = 'TEMPLATE_CATEGORY_MONGODB',
  MYSQL = 'TEMPLATE_CATEGORY_MYSQL',
  NODE = 'TEMPLATE_CATEGORY_NODE',
  POSTGRESQL = 'TEMPLATE_CATEGORY_POSTGRESQL',
  PROXYSQL = 'TEMPLATE_CATEGORY_PROXYSQL',
  VALKEY = 'TEMPLATE_CATEGORY_VALKEY',
  HAPROXY = 'TEMPLATE_CATEGORY_HAPROXY',
}

export interface BoolParamDefinition {
  default?: boolean;
}

export interface FloatParamDefinition {
  default?: number;
  min?: number;
  max?: number;
}

export interface StringParamDefinition {
  default?: string;
}

export interface ParamDefinition {
  name: string;
  summary: string;
  unit: ParamUnit;
  type: ParamType;
  bool?: BoolParamDefinition;
  float?: FloatParamDefinition;
  string?: StringParamDefinition;
}

export interface TemplateQuery {
  refId: string;
  expr: string;
}

export interface TemplateExpression {
  refId: string;
  // Expression type; currently only "math".
  type: string;
  // References other steps by ref ID, e.g. "$A > [[ .threshold ]]".
  expression: string;
}

export interface Template {
  name: string;
  summary: string;
  expr: string;
  params: ParamDefinition[];
  // Duration serialized by the JSON gateway as a string, e.g. "60s".
  for: string;
  severity: Severity;
  labels: Record<string, string>;
  annotations: Record<string, string>;
  source: TemplateSource;
  // RFC3339 timestamp; empty for built-in and SaaS templates.
  createdAt?: string;
  // YAML file content; empty for built-in and SaaS templates.
  yaml: string;
  // Query and expression steps for multi-expression templates. Both omitted for
  // single-expression templates, which use `expr` instead.
  queries?: TemplateQuery[];
  expressions?: TemplateExpression[];
  // Ref ID of the step used as the alert condition, e.g. "C".
  condition?: string;
  category: TemplateCategory;
}

export interface ListTemplatesParams {
  pageSize?: number;
  pageIndex?: number;
  reload?: boolean;
}

export interface ListTemplatesResponse {
  totalItems: number;
  totalPages: number;
  templates: Template[];
}

export interface CreateTemplatePayload {
  yaml: string;
}

export interface UpdateTemplatePayload {
  name: string;
  yaml: string;
}

export interface DeleteTemplatePayload {
  name: string;
}

export interface ParamValue {
  name: string;
  type: ParamType;
  bool?: boolean;
  float?: number;
  string?: string;
}

export interface Filter {
  type: FilterType;
  label: string;
  regexp: string;
}

export interface CreateRulePayload {
  templateName: string;
  name: string;
  group: string;
  folderUid: string;
  params: ParamValue[];
  // Duration string, e.g. "60s".
  for: string;
  severity: Severity;
  customLabels?: Record<string, string>;
  filters: Filter[];
  // Duration string, e.g. "60s".
  interval: string;
}
