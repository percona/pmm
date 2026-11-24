import type { AgentType } from 'types/agent.types';
import type { ServiceChanges } from 'types/services.types';
import type { EditServiceFormValues } from './EditService.schema';

// The server rejects mixing them, so the form offers one choice
export enum RdsAuthMode {
  accessKey = 'access_key',
  iamRole = 'iam_role',
  hostCredentials = 'host_credentials',
}

export enum QuerySource {
  none = 'none',
  perfschema = 'perfschema',
  slowlog = 'slowlog',
  profiler = 'profiler',
  mongolog = 'mongolog',
  pgstatements = 'pgstatements',
  pgstatmonitor = 'pgstatmonitor',
}

// The form fields that hold a service setting, named after the setting
export type SettingName = Exclude<
  keyof EditServiceFormValues,
  'rdsAuthMode' | 'awsAccessKey' | 'awsSecretKey' | 'awsRoleArn'
>;

export interface QuerySourceOption {
  value: QuerySource;
  agentType: AgentType;
  change: keyof ServiceChanges;
  // The QAN agent that adding a service on an RDS or Azure node creates, the
  // only one the server allows there
  isCloud?: boolean;
}
