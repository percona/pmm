import { act, render, screen } from '@testing-library/react';
import { useElapsedSeconds } from './useElapsedSeconds';

const START = '2026-09-07T10:00:00Z';

const Probe = ({
  startedAt,
  isRunning,
}: {
  startedAt?: string | null;
  isRunning: boolean;
}) => {
  const elapsed = useElapsedSeconds(startedAt, isRunning);
  return (
    <span data-testid="elapsed">{elapsed === null ? 'none' : elapsed}</span>
  );
};

const elapsedText = () => screen.getByTestId('elapsed').textContent;

describe('useElapsedSeconds', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-09-07T10:00:30Z'));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('reports the seconds since the start time on first render', () => {
    render(<Probe startedAt={START} isRunning />);

    expect(elapsedText()).toBe('30');
  });

  it('advances once a second while running', () => {
    render(<Probe startedAt={START} isRunning />);

    act(() => {
      vi.advanceTimersByTime(3000);
    });

    expect(elapsedText()).toBe('33');
  });

  it('stops ticking once the run is no longer running', () => {
    const { rerender } = render(<Probe startedAt={START} isRunning />);

    rerender(<Probe startedAt={START} isRunning={false} />);
    act(() => {
      vi.advanceTimersByTime(5000);
    });

    // The last observed value stays put rather than being zeroed, so a caller
    // swapping to the server's `duration` never flashes an empty cell.
    expect(elapsedText()).toBe('30');
  });

  it('does not tick when it never started running', () => {
    render(<Probe startedAt={START} isRunning={false} />);

    act(() => {
      vi.advanceTimersByTime(5000);
    });

    expect(elapsedText()).toBe('30');
  });

  it('recomputes immediately when the start time changes', () => {
    const { rerender } = render(<Probe startedAt={START} isRunning />);

    rerender(<Probe startedAt="2026-09-07T10:00:20Z" isRunning />);

    expect(elapsedText()).toBe('10');
  });

  it('returns null without a usable start time', () => {
    render(<Probe startedAt={null} isRunning />);
    expect(elapsedText()).toBe('none');

    render(<Probe startedAt="not-a-date" isRunning />);
    expect(screen.getAllByTestId('elapsed')[1]?.textContent).toBe('none');
  });
});
