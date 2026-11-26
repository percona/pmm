import { type Dump, DumpStatus } from 'types/dump.types';
import {
  formatDataWindow,
  formatDate,
  formatRelativeTime,
  formatTimeRange,
  getDumpArchiveUrl,
  isDumpCreating,
} from './DumpPage.utils';

const dump: Dump = {
  dumpId: 'dump-1',
  status: DumpStatus.Success,
  serviceNames: ['mysql'],
  startTime: '2026-07-10T08:00:00Z',
  endTime: '2026-07-10T10:00:00Z',
  createdAt: '2026-07-10T10:01:00Z',
  encrypted: false,
};

describe('DumpPage utils', () => {
  it('builds archive URLs for plain and encrypted dumps', () => {
    expect(getDumpArchiveUrl(dump)).toBe('/dump/dump-1.tar.gz');
    expect(getDumpArchiveUrl({ ...dump, encrypted: true })).toBe(
      '/dump/dump-1.tar.gz.enc'
    );
  });

  it('formats dates in the given timezone and time ranges', () => {
    expect(formatDate(dump.createdAt, 'Europe/Bratislava')).toBe(
      '2026-07-10 12:01:00'
    );
    expect(formatDate('', 'UTC')).toBe('N/A');
    expect(formatDate('not-a-date', 'UTC')).toBe('N/A');
    expect(formatTimeRange(dump)).toBe('2 hours');
  });
});

describe('formatDataWindow', () => {
  const window = {
    ...dump,
    startTime: '2026-10-26T18:39:00Z',
    endTime: '2026-10-27T06:39:00Z',
  };

  it('formats both ends in the given timezone', () => {
    expect(formatDataWindow(window, 'UTC')).toBe(
      'Oct 26, 18:39 – Oct 27, 06:39'
    );
  });

  it("uses the PMM timezone, not the browser's", () => {
    expect(formatDataWindow(window, 'America/New_York')).toBe(
      'Oct 26, 14:39 – Oct 27, 02:39'
    );
  });

  it('shows N/A for a missing or invalid end', () => {
    expect(formatDataWindow({ ...window, endTime: '' }, 'UTC')).toBe('N/A');
    expect(formatDataWindow({ ...window, startTime: 'nope' }, 'UTC')).toBe(
      'N/A'
    );
  });
});

describe('formatRelativeTime', () => {
  it('describes the past', () => {
    const now = new Date('2026-10-27T06:39:00Z');

    expect(formatRelativeTime('2026-10-27T03:39:00Z', now)).toBe('3 hours ago');
    expect(formatRelativeTime('', now)).toBe('N/A');
  });
});

describe('isDumpCreating', () => {
  it('is true only while in progress', () => {
    expect(isDumpCreating({ ...dump, status: DumpStatus.InProgress })).toBe(
      true
    );
    [DumpStatus.Success, DumpStatus.Error, DumpStatus.Unspecified].forEach(
      (status) => expect(isDumpCreating({ ...dump, status })).toBe(false)
    );
  });
});
