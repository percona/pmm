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

export interface CategorizedAdvisor {
  [category: string]: {
    [summary: string]: Advisor;
  };
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
