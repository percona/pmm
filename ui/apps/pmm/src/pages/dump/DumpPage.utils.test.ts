import { Dump, DumpStatus } from 'types/dump.types';
import {
  formatTimeRange,
  getDumpArchiveUrl,
  getStatusLabel,
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

  it('formats status and time range values', () => {
    expect(getStatusLabel(DumpStatus.InProgress)).toBe('In progress');
    expect(formatTimeRange(dump)).toBe('2 hours');
  });
});
