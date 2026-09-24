import { renderHook, waitFor } from '@testing-library/react';
import { GrafanaAlertRuleDefinition } from 'types/alerting.types';
import { useAlertValueThreshold } from './useAlertValueThreshold';
import { wrapWithQueryProvider } from 'utils/testUtils';

const { evalAlertQueries } = vi.hoisted(() => ({
  evalAlertQueries: vi.fn(),
}));

vi.mock('api/alerting', () => ({
  evalAlertQueries,
}));

const definition: GrafanaAlertRuleDefinition = {
  uid: 'rule-1',
  condition: 'A',
  data: [
    {
      refId: 'A',
      datasourceUid: 'prometheus',
      model: {
        refId: 'A',
        expr: 'mysql_connections > 80',
      },
    },
  ],
};

describe('useAlertValueThreshold', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    evalAlertQueries.mockResolvedValue({
      results: {
        A: {
          status: 200,
          frames: [
            {
              schema: {
                fields: [
                  { name: 'Time', type: 'time' },
                  {
                    name: 'Value',
                    type: 'number',
                    labels: { service_id: 'service-1' },
                  },
                ],
              },
              data: { values: [[1_752_147_200_000], [100]] },
            },
          ],
        },
      },
    });
  });

  it('evaluates the provided ruler definition', async () => {
    const { result } = renderHook(
      () =>
        useAlertValueThreshold(definition, {
          service_id: 'service-1',
        }),
      {
        wrapper: ({ children }) => wrapWithQueryProvider(children),
      }
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toMatchObject({
      value: 100,
      threshold: 80,
      breaching: true,
    });
    expect(evalAlertQueries).toHaveBeenCalledWith([
      expect.objectContaining({
        model: expect.objectContaining({
          expr: 'mysql_connections',
          instant: true,
          range: false,
        }),
      }),
    ]);
  });

  it('does not evaluate until a ruler definition is available', () => {
    const { result } = renderHook(() => useAlertValueThreshold(undefined, {}), {
      wrapper: ({ children }) => wrapWithQueryProvider(children),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(evalAlertQueries).not.toHaveBeenCalled();
  });
});
