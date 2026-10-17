import { DataSourcePlugin } from '@grafana/data';
import { PTSummaryDataSource } from './PTSummaryDataSource';
import { QueryEditor } from './query-editor';

export const plugin = new DataSourcePlugin(PTSummaryDataSource).setQueryEditor(
  QueryEditor
);
