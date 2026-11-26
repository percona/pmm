import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderInTheme, testTheme } from '../../../test-utils';
import DistributionBar from './distribution-bar';

const segments = [
  { id: 'cpu', label: 'CPU', value: 60 },
  { id: 'io', label: 'I/O', value: 30 },
  { id: 'lock', label: 'Lock', value: 6 },
  { id: 'other', label: 'Other', value: 4 },
];

const segmentBoxes = (container: HTMLElement) =>
  Array.from(container.querySelectorAll<HTMLElement>('[data-segment]'));

describe('DistributionBar', () => {
  it('sizes the parts so they fill the whole bar', () => {
    const { container } = renderInTheme(
      <DistributionBar segments={segments} />
    );

    const grows = segmentBoxes(container).map((box) =>
      Number(getComputedStyle(box).flexGrow)
    );
    expect(grows).toEqual([0.6, 0.3, 0.06, 0.04]);
    expect(grows.reduce((sum, grow) => sum + grow, 0)).toBeCloseTo(1);
  });

  it('labels the parts in the bar only when they are wide enough', () => {
    const { container } = renderInTheme(
      <DistributionBar segments={segments} />
    );

    const [cpu, io, lock, other] = segmentBoxes(container);
    expect(cpu).toHaveTextContent('60%');
    expect(io).toHaveTextContent('30%');
    expect(lock).toBeEmptyDOMElement();
    expect(other).toBeEmptyDOMElement();
  });

  it('lists every part with its share in the legend, and hides the bar', () => {
    renderInTheme(
      <DistributionBar segments={segments} ariaLabel="Query time" />
    );

    const legend = screen.getByRole('list', { name: 'Query time' });
    expect(
      within(legend)
        .getAllByRole('listitem')
        .map((item) => item.textContent)
    ).toEqual(['CPU 60%', 'I/O 30%', 'Lock 6%', 'Other 4%']);
    expect(legend.previousElementSibling).toHaveAttribute(
      'aria-hidden',
      'true'
    );
  });

  it('keeps the legend for screen readers when it is hidden', () => {
    renderInTheme(
      <DistributionBar segments={segments} hideLegend ariaLabel="Query time" />
    );

    expect(screen.getByRole('list', { name: 'Query time' })).toHaveStyle({
      position: 'absolute',
      width: '1px',
    });
  });

  it('falls back to the chart palette, and takes a color per part', () => {
    const { container } = renderInTheme(
      <DistributionBar
        segments={[
          { id: 'a', label: 'A', value: 1 },
          { id: 'b', label: 'B', value: 1, color: 'primary' },
          { id: 'c', label: 'C', value: 1 },
        ]}
      />
    );

    const [a, b, c] = segmentBoxes(container);
    expect(a).toHaveStyle({ backgroundColor: testTheme.palette.charts.chart1 });
    expect(b).toHaveStyle({ backgroundColor: testTheme.palette.primary.main });
    expect(c).toHaveStyle({ backgroundColor: testTheme.palette.charts.chart3 });
  });

  it('formats shares with the given formatter', () => {
    renderInTheme(
      <DistributionBar
        segments={segments.slice(0, 2)}
        valueFormatter={(share, segment) => `${segment.id}=${share.toFixed(2)}`}
      />
    );

    expect(screen.getAllByText('cpu=0.67')).toHaveLength(2);
  });

  it('skips parts with nothing in them in the bar but still lists them', () => {
    const { container } = renderInTheme(
      <DistributionBar
        segments={[...segments, { id: 'idle', label: 'Idle', value: 0 }]}
      />
    );

    expect(segmentBoxes(container)).toHaveLength(4);
    expect(screen.getAllByRole('listitem')).toHaveLength(5);
    expect(screen.getByText('0%')).toBeInTheDocument();
  });

  it('draws an empty track when every value is 0', () => {
    const { container } = renderInTheme(
      <DistributionBar segments={[{ id: 'a', label: 'A', value: 0 }]} />
    );

    expect(segmentBoxes(container)).toHaveLength(0);
    expect(container.querySelector('[aria-hidden="true"]')).toHaveStyle({
      backgroundColor: testTheme.palette.dividers.divider,
    });
  });
});
