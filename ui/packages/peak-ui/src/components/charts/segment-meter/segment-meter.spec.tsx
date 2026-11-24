import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderInTheme, testTheme } from '../../../test-utils';
import SegmentMeter from './segment-meter';

const filledCount = (container: HTMLElement) =>
  container.querySelectorAll('[data-filled="true"]').length;

describe('SegmentMeter', () => {
  it.each([
    [0, 0],
    [0.001, 1],
    [0.12, 1],
    [0.5, 3],
    [0.9, 4],
    [0.99, 4],
    [1, 5],
    [1.4, 5],
  ])('fills the dashes for a share of %s', (value, filled) => {
    const { container } = renderInTheme(<SegmentMeter value={value} />);

    expect(container.querySelectorAll('[data-filled]')).toHaveLength(5);
    expect(filledCount(container)).toBe(filled);
  });

  it('splits into the given number of dashes', () => {
    const { container } = renderInTheme(
      <SegmentMeter value={0.5} segments={10} />
    );

    expect(container.querySelectorAll('[data-filled]')).toHaveLength(10);
    expect(filledCount(container)).toBe(5);
  });

  it('exposes the share as a meter from 0 to 100', () => {
    renderInTheme(
      <SegmentMeter value={0.123} ariaLabel="Share of total load" />
    );

    const meter = screen.getByRole('meter', { name: 'Share of total load' });
    expect(meter).toHaveAttribute('aria-valuemin', '0');
    expect(meter).toHaveAttribute('aria-valuemax', '100');
    expect(meter).toHaveAttribute('aria-valuenow', '12.3');
    expect(meter).toHaveAttribute('aria-valuetext', '12.3% of total');
  });

  it('reads the given value text', () => {
    renderInTheme(<SegmentMeter value={0.5} valueText="half of all queries" />);

    expect(
      screen.getByRole('meter', { name: 'Share of total' })
    ).toHaveAttribute('aria-valuetext', 'half of all queries');
  });

  it('hides a decorative meter from assistive technology', () => {
    renderInTheme(<SegmentMeter value={0.5} decorative data-testid="meter" />);

    expect(screen.queryByRole('meter')).not.toBeInTheDocument();
    expect(screen.getByTestId('meter')).toHaveAttribute('aria-hidden', 'true');
  });

  it('colors filled dashes with the mark color and empty ones with the track', () => {
    const { container } = renderInTheme(
      <SegmentMeter value={0.2} color="chart6" />
    );

    expect(container.querySelector('[data-filled="true"]')).toHaveStyle({
      backgroundColor: testTheme.palette.charts.chart6,
    });
    expect(container.querySelector('[data-filled="false"]')).toHaveStyle({
      backgroundColor: testTheme.palette.dividers.divider,
    });
  });
});
