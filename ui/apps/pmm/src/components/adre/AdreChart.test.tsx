import { render, screen } from '@testing-library/react';
import { TestWrapper } from 'utils/testWrapper';
import AdreChart from './AdreChart';

const spec = {
  ds: 'Metrics',
  title: 'Handlers by service',
  from: '2026-10-19T20:00:00Z',
  to: '2026-10-19T21:00:00Z',
  unit: 'ops',
  legend: '{{service_name}} · {{handler}}',
  query: 'sum by (service_name, handler) (rate(x[5m]))',
};

const renderChart = (s = spec) =>
  render(
    <TestWrapper>
      <AdreChart spec={s} />
    </TestWrapper>
  );

describe('AdreChart', () => {
  it('embeds the Assistant Chart panel over the fixed window', () => {
    renderChart();

    const figure = screen.getByRole('figure', { name: 'Handlers by service' });
    expect(figure).toHaveTextContent('2026-10-19 20:00–21:00 UTC · Metrics');
    const src = new URL(
      screen.getByTitle('Handlers by service').getAttribute('src') ?? '',
      'https://pmm.test'
    );
    expect(src.pathname).toBe(
      '/graph/d-solo/pmm-assistant-chart/assistant-chart'
    );
    expect(src.searchParams.get('var-query')).toBe(spec.query);
    expect(src.searchParams.get('from')).toBe(String(Date.parse(spec.from)));
    expect(
      screen.getByRole('link', { name: 'Open in Explore' })
    ).toHaveAttribute('href', expect.stringContaining('/graph/explore?'));
  });

  it('says when the window cannot be read and shows the query', () => {
    renderChart({ ...spec, from: 'yesterday' });
    expect(
      screen.getByText(
        "Can't read the time range yesterday – 2026-10-19T21:00:00Z"
      )
    ).toBeInTheDocument();
    expect(screen.getByText(spec.query)).toBeInTheDocument();
    expect(screen.queryByRole('figure')).toBeNull();
  });
});
