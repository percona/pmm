import { renderHook, waitFor } from '@testing-library/react';
import { generateCsv, mkConfig } from 'export-to-csv';
import { api } from 'api/api';
import { useRealtimeQueries } from 'hooks/api/useRealtime';
import { wrapWithQueryProvider } from 'utils/testUtils';
import { collectCsvColumns, mapQueryToCsvRow } from './exportRtaQueriesToCsv';

vi.mock('contexts/user', () => ({
  useUser: () => ({ user: { id: 1 } }),
}));

// Exactly what the REST gateway puts on the wire, snake_case and all. The two
// `**_new_**` fields do not exist in the proto, the TypeScript types or the
// exporter: they stand in for a future backend release.
const WIRE_RESPONSE = {
  queries: [
    {
      service_id: 'service-1',
      service_name: 'Service 1',
      query_id: 'query-1',
      query_text: '{ find: "mycollection" }',
      query_raw_json: '{ find: "mycollection" }',
      query_execution_duration: '10s',
      query_collect_time: '2021-01-01T00:00:00Z',
      client_address: '127.0.0.1',
      mongo_db_payload: {
        db_instance_address: '127.0.0.1',
        client_app_name: 'client-app-name',
        database_name: 'database-name',
        operation_start_time: '2021-01-01T00:00:00Z',
        plan_summary: 'COLLSCAN',
        operation: 'find',
        username: 'username',
        collection: 'mycollection',
      },
    },
    {
      service_id: 'service-1',
      service_name: 'Service 1',
      query_id: 'query-2',
      query_text: '{ aggregate: "orders" }',
      query_raw_json: '{ aggregate: "orders" }',
      query_execution_duration: '2.5s',
      query_collect_time: '2021-01-01T00:00:05Z',
      client_address: '127.0.0.2',
      // a new top-level field shipped by a later backend
      read_preference: 'secondary',
      mongo_db_payload: {
        db_instance_address: '127.0.0.1',
        client_app_name: 'client-app-name',
        database_name: 'database-name',
        operation_start_time: '2021-01-01T00:00:05Z',
        plan_summary: 'IXSCAN',
        operation: 'aggregate',
        username: 'username',
        // collection is unset on this row, and a new nested field appears
        num_yields: 4,
      },
    },
  ],
};

describe('rta csv export, from the wire', () => {
  beforeEach(() => {
    api.defaults.adapter = async (config) => ({
      data: WIRE_RESPONSE,
      status: 200,
      statusText: 'OK',
      headers: {},
      config,
    });
  });

  it('exports fields the UI has never heard of, named as the API named them', async () => {
    const { result } = renderHook(
      () => useRealtimeQueries({ serviceIds: ['service-1'] }),
      { wrapper: ({ children }) => wrapWithQueryProvider(children) }
    );

    await waitFor(() => expect(result.current.data).toHaveLength(2));

    const rows = (result.current.data ?? []).map(mapQueryToCsvRow);
    const columns = collectCsvColumns(rows);
    const csv = String(generateCsv(mkConfig({ columnHeaders: columns }))(rows));

    // the agreed columns keep their names and their order
    expect(columns.slice(0, 14)).toEqual([
      'operation_id',
      'elapsed_exec_time_sec',
      'db_instance_address',
      'client_address',
      'database_name',
      'service',
      'user_name',
      'collection',
      'operation',
      'plan_summary',
      'client_app_name',
      'operation_start_time',
      'data_capture_time',
      'raw_query',
    ]);

    // fields the exporter has no knowledge of are carried through under the
    // name the API used on the wire
    expect(columns).toContain('read_preference');
    expect(columns).toContain('num_yields');
    expect(csv).toContain('"secondary"');
    expect(csv).toContain('4');

    // the duration is parsed to seconds, and only once
    expect(columns).not.toContain('query_execution_duration');
    expect(rows[0].elapsed_exec_time_sec).toBe(10);
    expect(rows[1].elapsed_exec_time_sec).toBe(2.5);

    // a column absent from the first row still survives, and vice versa
    expect(columns).toContain('collection');
    expect(rows[0].collection).toBe('mycollection');
    expect(rows[1].collection).toBeUndefined();
  });
});
