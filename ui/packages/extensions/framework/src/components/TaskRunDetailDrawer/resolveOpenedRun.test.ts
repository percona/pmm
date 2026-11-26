import type { TaskHistoryEntry } from '../../hooks';
import { resolveOpenedRun } from './resolveOpenedRun';

const row = (id: number | null, status: string) =>
  ({ id, status }) as unknown as TaskHistoryEntry;

describe('resolveOpenedRun', () => {
  it('returns nothing when no run is open', () => {
    expect(resolveOpenedRun(null, [row(1, 'success')])).toBeNull();
  });

  it('picks up a status change the clicked row predates', () => {
    // The point of the helper: a run watched in the drawer has to reach its
    // terminal status without the drawer being closed and reopened.
    const opened = row(1, 'running');

    const resolved = resolveOpenedRun(opened, [
      row(2, 'success'),
      row(1, 'failed'),
    ]);

    expect(resolved?.status).toBe('failed');
  });

  it('keeps the clicked row when it has paged out of the list', () => {
    const opened = row(1, 'running');

    expect(resolveOpenedRun(opened, [row(2, 'success')])).toBe(opened);
  });

  it('keeps the clicked row when there is no list yet', () => {
    const opened = row(1, 'running');

    expect(resolveOpenedRun(opened, undefined)).toBe(opened);
  });

  it('keeps a run with no id, which there is nothing to match on', () => {
    const opened = row(null, 'running');

    expect(resolveOpenedRun(opened, [row(1, 'failed')])).toBe(opened);
  });
});
