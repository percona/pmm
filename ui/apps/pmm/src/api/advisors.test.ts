import { afterEach, describe, it, expect } from 'vitest';
import type { InternalAxiosRequestConfig } from 'axios';
import {
  type AdvisorCheckInput,
  AdvisorInterval,
  AdvisorTechnology,
} from 'types/advisors.types';
import { api } from './api';
import {
  camelizeKeys,
  createAdvisorCheck,
  getAdvisorCheck,
  listInsights,
  testAdvisorCheck,
  updateAdvisorCheck,
} from './advisors';

describe('camelizeKeys', () => {
  it('camelizes schema field names', () => {
    expect(
      camelizeKeys({
        total_items: 2,
        results: [{ check_name: 'chk', read_more_url: 'https://example.com' }],
      })
    ).toEqual({
      totalItems: 2,
      results: [{ checkName: 'chk', readMoreUrl: 'https://example.com' }],
    });
  });

  it('leaves label keys exactly as the API returned them', () => {
    const labels = {
      service_name: 'mysql-svc',
      node_id: 'pmm-server',
      agent_type: 'qan-mysql-perfschema-agent',
      az: 'us-east-1f',
      myCustomLabel: 'kept',
    };

    const result = camelizeKeys({
      results: [{ check_name: 'chk', labels }],
    }) as { results: Array<{ labels: Record<string, string> }> };

    expect(result.results[0].labels).toEqual(labels);
  });

  it('leaves query parameter keys exactly as the API returned them', () => {
    expect(
      camelizeKeys({
        check: {
          user_defined: true,
          queries: [
            {
              type: 'POSTGRESQL_SELECT',
              query: 'SELECT 1',
              parameters: { all_dbs: 'true' },
            },
          ],
        },
      })
    ).toEqual({
      check: {
        userDefined: true,
        queries: [
          {
            type: 'POSTGRESQL_SELECT',
            query: 'SELECT 1',
            parameters: { all_dbs: 'true' },
          },
        ],
      },
    });
  });

  it('passes through primitives and nulls', () => {
    expect(camelizeKeys(null)).toBeNull();
    expect(camelizeKeys('a_b')).toBe('a_b');
    expect(camelizeKeys(7)).toBe(7);
  });
});

describe('advisors API wire format', () => {
  const originalAdapter = api.defaults.adapter;

  const CHECK_RESPONSE = {
    check: {
      name: 'custom_pg_check',
      user_defined: true,
      disabled_service_ids: ['svc-1'],
      queries: [
        {
          type: 'POSTGRESQL_SELECT',
          query: 'SELECT 1',
          parameters: { all_dbs: 'true' },
        },
      ],
    },
  };

  const CHECK_INPUT: AdvisorCheckInput = {
    name: 'custom_pg_check',
    summary: 'Summary',
    description: 'Description',
    category: 'Configuration',
    technology: AdvisorTechnology.postgresql,
    interval: AdvisorInterval.standard,
    queries: [
      {
        type: 'POSTGRESQL_SELECT',
        query: 'SELECT 1',
        // a name the client's own snake-casing would rewrite
        parameters: { all_dbs: 'true', allDbs: 'kept' },
      },
    ],
    script: 'print(1)',
  };

  const stubApi = (body: unknown) => {
    const calls: InternalAxiosRequestConfig[] = [];
    api.defaults.adapter = async (config) => {
      calls.push(config);
      return {
        data: JSON.stringify(body),
        status: 200,
        statusText: 'OK',
        headers: {},
        config,
      };
    };
    return calls;
  };

  const sentBody = (config: InternalAxiosRequestConfig) =>
    JSON.parse(config.data as string);

  afterEach(() => {
    api.defaults.adapter = originalAdapter;
  });

  it('keeps parameter keys of a fetched check as stored', async () => {
    stubApi(CHECK_RESPONSE);

    expect(await getAdvisorCheck('custom_pg_check')).toEqual({
      name: 'custom_pg_check',
      userDefined: true,
      disabledServiceIds: ['svc-1'],
      queries: [
        {
          type: 'POSTGRESQL_SELECT',
          query: 'SELECT 1',
          parameters: { all_dbs: 'true' },
        },
      ],
    });
  });

  it('sends parameter keys untouched when creating and updating a check', async () => {
    const calls = stubApi(CHECK_RESPONSE);

    const created = await createAdvisorCheck(CHECK_INPUT);
    await updateAdvisorCheck('custom_pg_check', CHECK_INPUT);

    expect(calls).toHaveLength(2);
    for (const call of calls) {
      expect(call.headers.getContentType()).toBe('application/json');
      expect(sentBody(call)).toEqual({ check: CHECK_INPUT });
    }
    expect(created.queries?.[0].parameters).toEqual({ all_dbs: 'true' });
  });

  it('snake-cases the rest of a test request', async () => {
    const calls = stubApi({ results: [], script_output: 'out' });

    const response = await testAdvisorCheck({
      check: CHECK_INPUT,
      serviceId: 'svc-1',
    });

    expect(sentBody(calls[0])).toEqual({
      check: CHECK_INPUT,
      service_id: 'svc-1',
    });
    expect(response).toEqual({ results: [], scriptOutput: 'out' });
  });

  it('keeps insight labels as stored', async () => {
    const calls = stubApi({
      total_items: 1,
      results: [{ check_name: 'chk', labels: { service_name: 'mysql-svc' } }],
    });

    const response = await listInsights({ pageSize: 10, checkName: 'chk' });

    // query params are still snake-cased by the client's interceptor
    expect(calls[0].params).toEqual({ page_size: 10, check_name: 'chk' });
    expect(calls[0].data).toBeUndefined();
    expect(response).toEqual({
      totalItems: 1,
      results: [{ checkName: 'chk', labels: { service_name: 'mysql-svc' } }],
    });
  });
});
