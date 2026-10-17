import type { DataQuery } from '@grafana/schema';

export enum DatasourceType {
  postgresql = 'postgresql',
  mongodb = 'mongodb',
  mysql = 'mysql',
  node = 'node',
}

export interface PTSummaryQueryType {
  type?: DatasourceType;
  variableName?: string;
}

// Saved dashboards store an object in `queryType`, which DataQuery types as a string.
export type PTSummaryQuery = Omit<DataQuery, 'queryType'> & {
  queryType?: PTSummaryQueryType;
};

export const toPTSummaryQuery = (query: DataQuery) =>
  query as unknown as PTSummaryQuery;
