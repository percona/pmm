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

  it('shows the total of each severity in the legend', () => {
    render(
      <TestWrapper>
        <LogsVolumeChart buckets={buckets} step="60s" onZoom={vi.fn()} />
      </TestWrapper>
    );
    expect(screen.getByTestId('logs-volume-total-error')).toHaveTextContent(
      'Error 28'
    );
    expect(screen.getByTestId('logs-volume-total-warning')).toHaveTextContent(
      'Warning 8'
    );
    expect(screen.getByTestId('logs-volume-total-info')).toHaveTextContent(
      'Info 80'
    );
    expect(screen.getByTestId('logs-volume-total-other')).toHaveTextContent(
      'Other 0'
    );
  });

  it('labels the count axis with round numbers', () => {
    // The tallest bucket holds 7 + 1 + 10 = 18 lines.
    render(
      <TestWrapper>
        <LogsVolumeChart buckets={buckets} step="60s" onZoom={vi.fn()} />
      </TestWrapper>
    );
    expect(
      screen.getAllByTestId('logs-volume-count-label').map((l) => l.textContent)
    ).toEqual(['20', '10', '0']);
  });

  it('labels the time axis with bucket start times', () => {
    render(
      <TestWrapper>
        <LogsVolumeChart buckets={buckets} step="60s" onZoom={vi.fn()} />
      </TestWrapper>
    );
    const ticks = screen.getAllByTestId('logs-volume-time-label');
    expect(ticks.length).toBeGreaterThan(2);
    expect(ticks[0]).toHaveTextContent(
      new Date(start).toLocaleTimeString(undefined, {
        hour: '2-digit',
        minute: '2-digit',
      })
    );
  });
});
