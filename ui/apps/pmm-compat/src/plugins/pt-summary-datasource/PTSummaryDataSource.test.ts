import {
  FieldType,
  toDataFrame,
  type DataQueryRequest,
  type DataSourceInstanceSettings,
} from '@grafana/data';
import {
  startNodeAction,
  startServiceAction,
} from 'plugins/shared/api/actions';
import { getActionResult } from 'plugins/shared/api/actions.utils';
import { DatasourceType } from './PTSummary.types';
import { PTSummaryDataSource } from './PTSummaryDataSource';

const replaceMock = jest.fn((value: string) => value);

jest.mock('@grafana/runtime', () => ({
  getTemplateSrv: () => ({ replace: replaceMock }),
}));
jest.mock('plugins/shared/api/actions');
jest.mock('plugins/shared/api/actions.utils');

const startNodeActionMock = jest.mocked(startNodeAction);
const startServiceActionMock = jest.mocked(startServiceAction);
const getActionResultMock = jest.mocked(getActionResult);

const query = (type?: DatasourceType, variableName?: string) =>
  ({
    targets: [{ refId: 'A', queryType: { type, variableName } }],
  }) as unknown as DataQueryRequest;

const frame = (value: string) => ({
  data: [
    toDataFrame({
      fields: [{ name: 'summary', values: [value], type: FieldType.string }],
    }),
  ],
});

describe('PTSummaryDataSource::', () => {
  const datasource = new PTSummaryDataSource({} as DataSourceInstanceSettings);

  beforeEach(() => {
    jest.clearAllMocks();
    getActionResultMock.mockImplementation(async (actionId) => ({
      value: actionId,
      error: '',
    }));
  });

  it('should return the node summary', async () => {
    startNodeActionMock.mockResolvedValue({
      action_id: 'node',
      pmm_agent_id: 'agent',
    });

    expect(await datasource.query(query(DatasourceType.node))).toEqual(
      frame('node')
    );
    expect(startNodeActionMock).toHaveBeenCalledWith({ node_id: '$node_id' });
  });

  it('should fall back to the node summary for an unknown type', async () => {
    startNodeActionMock.mockResolvedValue({
      action_id: 'node',
      pmm_agent_id: 'agent',
    });

    expect(await datasource.query(query())).toEqual(frame('node'));
  });

  it.each([
    [DatasourceType.mysql, 'pt_mysql_summary', '$service_name'],
    [DatasourceType.postgresql, 'pt_postgres_summary', '$service_name'],
    [DatasourceType.mongodb, 'pt_mongodb_summary', '$service-name'],
  ])('should return the %s summary', async (type, key, serviceVariable) => {
    startServiceActionMock.mockResolvedValue({
      [key]: { action_id: type, pmm_agent_id: 'agent' },
    });

    expect(await datasource.query(query(type))).toEqual(frame(type));
    expect(startServiceActionMock).toHaveBeenCalledWith({
      [key]: { service_id: serviceVariable },
    });
  });

  it('should use the selected variable name', async () => {
    startNodeActionMock.mockResolvedValue({
      action_id: 'node',
      pmm_agent_id: 'agent',
    });

    await datasource.query(query(DatasourceType.node, 'custom'));

    expect(replaceMock).toHaveBeenCalledWith('$custom');
  });

  it('should return the action error', async () => {
    startNodeActionMock.mockResolvedValue({
      action_id: 'node',
      pmm_agent_id: 'agent',
    });
    getActionResultMock.mockResolvedValue({ value: '', error: 'failed' });

    expect(await datasource.query(query(DatasourceType.node))).toEqual(
      frame('failed')
    );
  });

  it('should return the API error message', async () => {
    startNodeActionMock.mockRejectedValue({ data: { message: 'not found' } });

    expect(await datasource.query(query(DatasourceType.node))).toEqual(
      frame('not found')
    );
  });

  it('should return an error when the action did not start', async () => {
    startServiceActionMock.mockResolvedValue({});

    expect(await datasource.query(query(DatasourceType.mysql))).toEqual(
      frame('error')
    );
  });
});
