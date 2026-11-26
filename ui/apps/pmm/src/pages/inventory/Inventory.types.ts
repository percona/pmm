export enum MonitoringStatus {
  ok = 'OK',
  failed = 'Failed',
  warning = 'Warning',
}

export type InventoryTab = 'services' | 'nodes';

export type InventoryEntity = InventoryTab | 'agents';
