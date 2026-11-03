import {
  DataSourceApi,
  FieldType,
  toDataFrame,
  type DataQueryRequest,
  type DataQueryResponse,
  type DataSourceInstanceSettings,
} from '@grafana/data';
import type { FetchError } from '@grafana/runtime';
import { getActionResult } from 'plugins/shared/api/actions.utils';
import type { ApiErrorResponse } from 'plugins/shared/types/api.types';
import { SUMMARY_FIELD_NAME } from './PTSummary.constants';
import { Messages } from './PTSummary.messages';
import { startPTSummaryAction } from './PTSummary.service';
import { toPTSummaryQuery } from './PTSummary.types';

export class PTSummaryDataSource extends DataSourceApi {
  constructor(instanceSettings: DataSourceInstanceSettings) {
    super(instanceSettings);
  }

  async query(options: DataQueryRequest): Promise<DataQueryResponse> {
    const [target] = options.targets;
    const queryType = target ? toPTSummaryQuery(target).queryType : undefined;

    try {
      const action = await startPTSummaryAction(
        queryType?.type,
        queryType?.variableName
      );

      if (!action) {
        return this.newDataFrame(Messages.error);
      }

      const result = await getActionResult(action.action_id);

      return this.newDataFrame(result.value || result.error);
    } catch (e) {
      const { data } = e as FetchError<ApiErrorResponse>;

      return this.newDataFrame(data?.message || Messages.error);
    }
  }

  async testDatasource() {
    return {
      status: 'success',
      message: Messages.success,
    };
  }

  newDataFrame(value: string) {
    return {
      data: [
        toDataFrame({
          fields: [
            {
              name: SUMMARY_FIELD_NAME,
              values: [value],
              type: FieldType.string,
            },
          ],
        }),
      ],
    };
  }
}
