import { describe, it, expect } from 'vitest';
import { camelizeInsights } from './advisors';

describe('camelizeInsights', () => {
  it('camelizes schema field names', () => {
    expect(
      camelizeInsights({
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

    const result = camelizeInsights({
      results: [{ check_name: 'chk', labels }],
    }) as { results: Array<{ labels: Record<string, string> }> };

    expect(result.results[0].labels).toEqual(labels);
  });

  it('passes through primitives and nulls', () => {
    expect(camelizeInsights(null)).toBeNull();
    expect(camelizeInsights('a_b')).toBe('a_b');
    expect(camelizeInsights(7)).toBe(7);
  });
});
