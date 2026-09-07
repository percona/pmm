import { formatDuration } from './formatDuration';

describe('formatDuration', () => {
  it('renders an absent duration as an em-dash', () => {
    // The wire leaves `duration` null until a run finishes.
    expect(formatDuration(null)).toBe('—');
    expect(formatDuration(undefined)).toBe('—');
    expect(formatDuration(Number.NaN)).toBe('—');
  });

  it('keeps a decimal below a minute', () => {
    expect(formatDuration(0)).toBe('0.0s');
    expect(formatDuration(2.44)).toBe('2.4s');
    expect(formatDuration(59.9)).toBe('59.9s');
  });

  it('drops the decimal from a minute up', () => {
    expect(formatDuration(60)).toBe('1m 0s');
    expect(formatDuration(127.4)).toBe('2m 7s');
    expect(formatDuration(3599)).toBe('59m 59s');
  });

  it('buckets an hour or more into hours and minutes', () => {
    expect(formatDuration(3600)).toBe('1h 0m');
    expect(formatDuration(6420)).toBe('1h 47m');
    // The unbucketed minutes format this replaces read '120m 0s' here.
    expect(formatDuration(7200)).toBe('2h 0m');
  });

  it('clamps a negative duration to zero', () => {
    // A client clock behind the server's can subtract past a run's start.
    expect(formatDuration(-4)).toBe('0.0s');
  });
});
