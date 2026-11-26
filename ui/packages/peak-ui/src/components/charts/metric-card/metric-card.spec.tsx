import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderInTheme } from '../../../test-utils';
import MetricCard from './metric-card';

const data = [5, 3, 8].map((value, index) => ({
  timestamp: index * 60_000,
  value,
}));
const values = [
  { label: 'Per second', value: '1.4 ms' },
  { label: 'Sum', value: '2.1 h', note: '34%' },
  { label: 'Per query', value: '0.8 ms' },
];

describe('MetricCard', () => {
  it('is a group named by its title, with each value under its label', () => {
    renderInTheme(
      <MetricCard
        title="Query time"
        values={values}
        sparkline={{ data, width: 200 }}
      />
    );

    expect(screen.getByRole('group', { name: 'Query time' })).toHaveAttribute(
      'aria-busy',
      'false'
    );
    expect(screen.getAllByRole('term').map((term) => term.textContent)).toEqual(
      ['Per second', 'Sum', 'Per query']
    );
    expect(
      screen.getAllByRole('definition').map((value) => value.textContent)
    ).toEqual(['1.4 ms', '2.1 h34%', '0.8 ms']);
  });

  it('draws the trend as a bar sparkline by default', () => {
    const { container } = renderInTheme(
      <MetricCard
        title="Query time"
        values={values}
        sparkline={{ data, width: 200 }}
      />
    );

    expect(
      screen.getByRole('group', { name: 'Latest 8, lowest 3, highest 8' })
    ).toBeInTheDocument();
    expect(container.querySelector('rect, path')).toBeInTheDocument();
  });

  it('shows placeholders and reports busy while loading', () => {
    const { container } = renderInTheme(
      <MetricCard
        title="Rows sent"
        values={values}
        sparkline={{ data }}
        loading
      />
    );

    expect(screen.getByRole('group', { name: 'Rows sent' })).toHaveAttribute(
      'aria-busy',
      'true'
    );
    expect(screen.queryByText('1.4 ms')).not.toBeInTheDocument();
    expect(container.querySelectorAll('.MuiSkeleton-root')).toHaveLength(4);
  });

  it('shows a note of 0 as a note', () => {
    renderInTheme(
      <MetricCard
        title="Errors"
        values={[{ label: 'Sum', value: '12', note: 0 }]}
      />
    );

    expect(screen.getByText('0')).toHaveClass('MuiTypography-caption');
  });

  it('says there is no data when the trend has no points', () => {
    renderInTheme(
      <MetricCard title="Lock time" values={values} sparkline={{ data: [] }} />
    );

    expect(screen.getByText('No data')).toBeInTheDocument();
  });
});
