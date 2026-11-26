import { createRef } from 'react';
import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderInTheme } from '../../../test-utils';
import StatTile from './stat-tile';

const data = [1, 4, 2].map((value, index) => ({
  timestamp: index * 60_000,
  value,
}));

describe('StatTile', () => {
  it('is a group named by its label, with the value as text', () => {
    renderInTheme(<StatTile label="Operations" value="1.2K ops/s" />);

    const tile = screen.getByRole('group', { name: 'Operations' });
    expect(tile).toHaveTextContent('Operations1.2K ops/s');
    expect(tile).toHaveAttribute('aria-busy', 'false');
  });

  it('draws the trend as an area sparkline by default', () => {
    const { container } = renderInTheme(
      <StatTile label="Latency" value="3 ms" sparkline={{ data, width: 120 }} />
    );

    expect(
      screen.getByRole('group', { name: 'Latest 2, lowest 1, highest 4' })
    ).toBeInTheDocument();
    expect(container.querySelector('path')).toBeInTheDocument();
  });

  it('shows placeholders and reports busy while loading', () => {
    const { container } = renderInTheme(
      <StatTile label="Errors" value="0" sparkline={{ data }} loading />
    );

    expect(screen.getByRole('group', { name: 'Errors' })).toHaveAttribute(
      'aria-busy',
      'true'
    );
    expect(screen.queryByText('0')).not.toBeInTheDocument();
    expect(container.querySelectorAll('.MuiSkeleton-root')).toHaveLength(2);
  });

  it('keeps the placeholder at the height given for the trend', () => {
    const { container } = renderInTheme(
      <StatTile
        label="Errors"
        value="0"
        sparkline={{ data, height: 32 }}
        loading
      />
    );

    expect(container.querySelector('.MuiSkeleton-rounded')).toHaveStyle({
      height: '32px',
    });
  });

  it('draws a smaller value and a shorter trend at the small size', () => {
    const { container } = renderInTheme(
      <StatTile label="p99 read" value="38 ms" size="small" loading />
    );

    expect(screen.getByRole('group', { name: 'p99 read' })).toHaveStyle({
      padding: '12px',
    });
    expect(container.querySelector('.MuiTypography-h6')).toBeInTheDocument();
    expect(container.querySelector('.MuiSkeleton-rounded')).toHaveStyle({
      height: '34px',
    });
  });

  it('says there is no data when the trend has no points', () => {
    renderInTheme(
      <StatTile label="Saturation" value="—" sparkline={{ data: [] }} />
    );

    expect(screen.getByText('No data')).toBeInTheDocument();
    expect(screen.getAllByRole('group')).toHaveLength(1);
  });

  it('forwards the ref and passes other props to the Paper', () => {
    const ref = createRef<HTMLDivElement>();
    renderInTheme(
      <StatTile ref={ref} label="Operations" value="1" id="ops" sx={{ p: 1 }} />
    );

    expect(ref.current).toBe(screen.getByRole('group', { name: 'Operations' }));
    expect(ref.current).toHaveAttribute('id', 'ops');
    expect(ref.current).toHaveStyle({ padding: '8px', display: 'flex' });
  });
});
