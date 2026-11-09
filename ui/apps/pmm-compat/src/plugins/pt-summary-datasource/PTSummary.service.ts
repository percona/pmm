import { getTemplateSrv } from '@grafana/runtime';
import {
  startNodeAction,
  startServiceAction,
} from 'plugins/shared/api/actions';
import type { ActionStarted } from 'plugins/shared/types/actions.types';
import { DEFAULT_VARIABLE_NAMES } from './PTSummary.constants';
import { DatasourceType } from './PTSummary.types';

const SERVICE_ACTION_KEYS = {
  [DatasourceType.mysql]: 'pt_mysql_summary',
  [DatasourceType.postgresql]: 'pt_postgres_summary',
  [DatasourceType.mongodb]: 'pt_mongodb_summary',
} as const;

const resolveVariable = (type: DatasourceType, variableName?: string) =>
  getTemplateSrv().replace(`$${variableName || DEFAULT_VARIABLE_NAMES[type]}`);

export const startPTSummaryAction = async (
  type: DatasourceType = DatasourceType.node,
  variableName?: string
): Promise<ActionStarted | undefined> => {
  const key = SERVICE_ACTION_KEYS[type as keyof typeof SERVICE_ACTION_KEYS];

  if (!key) {
    return startNodeAction({
      node_id: resolveVariable(DatasourceType.node, variableName),
    });
  }

  const response = await startServiceAction({
    [key]: { service_id: resolveVariable(type, variableName) },
  });

  return response[key];
};
