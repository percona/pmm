import { DatasourceType } from './PTSummary.types';

export const DEFAULT_VARIABLE_NAMES: Record<DatasourceType, string> = {
  [DatasourceType.node]: 'node_id',
  [DatasourceType.mysql]: 'service_name',
  [DatasourceType.postgresql]: 'service_name',
  [DatasourceType.mongodb]: 'service-name',
};

export const SUMMARY_FIELD_NAME = 'summary';
