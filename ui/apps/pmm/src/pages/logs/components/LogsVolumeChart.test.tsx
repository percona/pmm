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

  it('selects a severity class from the legend', () => {
    const onSelectClass = vi.fn();
    render(
      <TestWrapper>
        <LogsVolumeChart
          buckets={buckets}
          step="60s"
          onZoom={vi.fn()}
          selectedClass="warning"
          onSelectClass={onSelectClass}
        />
      </TestWrapper>
    );
    expect(screen.getByTestId('logs-volume-total-warning')).toHaveAttribute(
      'aria-pressed',
      'true'
    );
    expect(screen.getByTestId('logs-volume-total-error')).toHaveAttribute(
      'aria-pressed',
      'false'
    );
    fireEvent.click(screen.getByTestId('logs-volume-total-error'));
    expect(onSelectClass).toHaveBeenCalledWith('error');
  });

  it("overlays the focused line's pattern and clears it from its chip", () => {
    const onClear = vi.fn();
    render(
      <TestWrapper>
        <LogsVolumeChart
          buckets={buckets}
          step="60s"
          onZoom={vi.fn()}
          overlay={{
            counts: new Map([[buckets[3].start, 4]]),
            label: 'deadlock detected',
            markerMs: start + 3 * 60_000 + 30_000,
            onClear,
          }}
        />
      </TestWrapper>
    );
    expect(screen.getByTestId('logs-volume-overlay')).toBeInTheDocument();
    expect(screen.getByTestId('logs-volume-marker')).toHaveStyle({
      left: '43.75%',
    });
    const chip = screen.getByTestId('logs-volume-overlay-chip');
    expect(chip).toHaveTextContent('Showing: deadlock detected');
    fireEvent.click(screen.getByTestId('CancelIcon'));
    expect(onClear).toHaveBeenCalled();
  });

  it('collapses to the legend and expands again', () => {
    render(
      <TestWrapper>
        <LogsVolumeChart buckets={buckets} step="60s" onZoom={vi.fn()} />
      </TestWrapper>
    );
    fireEvent.click(screen.getByRole('button', { name: 'Hide chart' }));
    expect(screen.queryAllByTestId('logs-volume-bucket')).toHaveLength(0);
    expect(screen.getByTestId('logs-volume-total-error')).toBeVisible();
    fireEvent.click(screen.getByRole('button', { name: 'Show chart' }));
    expect(screen.getAllByTestId('logs-volume-bucket')).toHaveLength(8);
  });
});
