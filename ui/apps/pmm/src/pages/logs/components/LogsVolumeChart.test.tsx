import { fireEvent, render, screen } from '@testing-library/react';
import { TestWrapper } from 'utils/testWrapper';
import { LogsVolumeChart } from './LogsVolumeChart';

const start = Date.parse('2026-10-17T10:00:00Z');
const buckets = Array.from({ length: 8 }, (_, i) => ({
  start: new Date(start + i * 60_000).toISOString(),
  error: i,
  warning: 1,
  info: 10,
  other: 0,
}));

describe('LogsVolumeChart', () => {
  it('renders one bar group per bucket', () => {
    render(
      <TestWrapper>
        <LogsVolumeChart buckets={buckets} step="60s" onZoom={vi.fn()} />
      </TestWrapper>
    );
    expect(screen.getAllByTestId('logs-volume-bucket')).toHaveLength(8);
  });

  it('zooms to the buckets dragged across', () => {
    const onZoom = vi.fn();
    render(
      <TestWrapper>
        <LogsVolumeChart buckets={buckets} step="60s" onZoom={onZoom} />
      </TestWrapper>
    );
    const bars = screen.getAllByTestId('logs-volume-bucket');
    fireEvent.mouseDown(bars[2]);
    fireEvent.mouseEnter(bars[5]);
    fireEvent.mouseUp(bars[5]);
    expect(onZoom).toHaveBeenCalledWith(start + 2 * 60_000, start + 6 * 60_000);
  });
});
