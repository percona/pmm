import { render, screen } from '@testing-library/react';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithQueryProvider } from 'utils/testUtils';
import AdreChart from './AdreChart';

const { queryMock } = vi.hoisted(() => ({ queryMock: vi.fn() }));

vi.mock('api/grafanaQuery', async (importOriginal) => ({
  ...(await importOriginal<typeof import('api/grafanaQuery')>()),
  getGrafanaDatasources: () =>
    Promise.resolve({
      Metrics: { uid: 'PA58DA793C7250F1B', type: 'prometheus' },
    }),
  queryGrafanaDatasource: queryMock,
}));

const spec = {
  ds: 'Metrics',
  title: 'CPU busy',
  from: '2026-10-19T20:00:00Z',
  to: '2026-10-19T21:00:00Z',
  unit: 'percentunit',
  query: 'sum by (node_name) (rate(node_cpu_seconds_total[5m]))',
};

const renderChart = (s = spec) =>
  render(
    wrapWithQueryProvider(
      <TestWrapper>
        <AdreChart spec={s} />
      </TestWrapper>
    )
  );

describe('AdreChart', () => {
  beforeEach(() => vi.clearAllMocks());

  it('runs the query over the fixed window and names each series', async () => {
    queryMock.mockResolvedValue([
      {
        schema: {
          fields: [
            { name: 'Time', type: 'time' },
            { name: 'Value', type: 'number', labels: { node_name: 'ps-1' } },
          ],
        },
        data: {
          values: [
            [
              Date.parse('2026-10-19T20:00:00Z'),
              Date.parse('2026-10-19T20:01:00Z'),
            ],
            [0.2, 0.4],
          ],
        },
      },
    ]);
    renderChart();

    const figure = screen.getByRole('figure', { name: 'CPU busy' });
    expect(figure).toHaveTextContent('2026-10-19 20:00–21:00 UTC · Metrics');
    expect(await screen.findByText('ps-1')).toBeInTheDocument();
    expect(queryMock).toHaveBeenCalledWith(
      { uid: 'PA58DA793C7250F1B', type: 'prometheus' },
      spec.query,
      {
        from: Date.parse('2026-10-19T20:00:00Z'),
        to: Date.parse('2026-10-19T21:00:00Z'),
      },
      300
    );
    expect(
      screen.getByRole('link', { name: 'Open in Explore' })
    ).toHaveAttribute('href', expect.stringContaining('/graph/explore?'));
  });

  it('says what went wrong and shows the query', async () => {
    renderChart({ ...spec, ds: 'Loki' });
    expect(
      await screen.findByText('No datasource named Loki')
    ).toBeInTheDocument();
    expect(screen.getByText(spec.query)).toBeInTheDocument();
    expect(queryMock).not.toHaveBeenCalled();
  });

  it('says when the window has no data', async () => {
    queryMock.mockResolvedValue([]);
    renderChart();
    expect(
      await screen.findByText('The query returned no data for this window.')
    ).toBeInTheDocument();
  });
});
