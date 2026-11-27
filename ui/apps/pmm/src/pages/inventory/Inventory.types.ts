export enum MonitoringStatus {
  ok = 'OK',
  failed = 'Failed',
  warning = 'Warning',
}

export type InventoryTab = 'services' | 'nodes';

export type InventoryEntity = InventoryTab | 'agents';

// How an RDS exporter or discovery authenticates to AWS; the server rejects mixing them
export enum RdsAuthMode {
  accessKey = 'access_key',
  iamRole = 'iam_role',
  hostCredentials = 'host_credentials',
}
