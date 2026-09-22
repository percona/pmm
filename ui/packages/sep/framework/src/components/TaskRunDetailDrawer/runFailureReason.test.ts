import type { TaskHistoryEntry } from '../../hooks';
import { firstLine, runFailureReason } from './runFailureReason';

const entry = (overrides: Record<string, unknown>) =>
  overrides as unknown as TaskHistoryEntry;

describe('runFailureReason', () => {
  it('returns null for the payload the wire carries today', () => {
    // TaskHistoryResponse has no error field until SEP-2000 lands, so every
    // candidate is absent and the UI falls back to pointing at the log.
    expect(
      runFailureReason(
        entry({
          status: 'failed',
          has_logs: true,
          execution_request: { tracking: {} },
        })
      )
    ).toBeNull();
  });

  it('returns null without a run', () => {
    expect(runFailureReason(null)).toBeNull();
    expect(runFailureReason(undefined)).toBeNull();
  });

  it('reads a reason off the row', () => {
    expect(runFailureReason(entry({ error_message: 'disk full' }))).toBe(
      'disk full'
    );
  });

  it('reads a reason out of the tracking bag', () => {
    // The untyped escape hatch the backend already uses for per-run
    // bookkeeping, and the likely first home for a reason.
    expect(
      runFailureReason(
        entry({
          execution_request: { tracking: { failure_reason: 'timeout' } },
        })
      )
    ).toBe('timeout');
  });

  it('prefers the row over the tracking bag', () => {
    expect(
      runFailureReason(
        entry({
          error: 'from row',
          execution_request: { tracking: { error: 'from tracking' } },
        })
      )
    ).toBe('from row');
  });

  it('prefers the most specific name when several are populated', () => {
    expect(
      runFailureReason(entry({ error: 'generic', error_message: 'specific' }))
    ).toBe('specific');
  });

  it('ignores a blank reason', () => {
    expect(runFailureReason(entry({ error_message: '   ' }))).toBeNull();
    expect(runFailureReason(entry({ error_message: 42 }))).toBeNull();
  });

  it('tolerates a null tracking bag', () => {
    expect(
      runFailureReason(entry({ execution_request: { tracking: null } }))
    ).toBeNull();
    expect(runFailureReason(entry({}))).toBeNull();
  });
});

describe('firstLine', () => {
  it('keeps a single-line reason whole', () => {
    expect(firstLine('disk full')).toBe('disk full');
  });

  it('takes only the first line of a traceback', () => {
    expect(firstLine('disk full\n  at /var/backups\n  at main')).toBe(
      'disk full'
    );
  });

  it('returns null for an absent or empty-first-line reason', () => {
    expect(firstLine(null)).toBeNull();
    expect(firstLine('\nreason on the second line')).toBeNull();
  });
});
