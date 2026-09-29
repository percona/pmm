import { describe, it, expect } from 'vitest';
import { GetUpdatesResponse } from 'types/updates.types';
import { formatCheckDate, getCheckStatus } from './Footer.utils';
import { Messages } from './Footer.messages';

const versionInfo = (lastCheck: string | null): GetUpdatesResponse => ({
  lastCheck,
  latest: null,
  installed: {
    version: '3.10.0',
    fullVersion: '3.10.0',
    timestamp: '2024-07-30T00:00:00Z',
  },
  latestNewsUrl: '',
  updateAvailable: false,
});

describe('formatCheckDate', () => {
  it('formats a timestamp as UTC', () => {
    expect(formatCheckDate('2024-07-30T10:34:05.886739003Z')).toBe(
      'July 30, 2024, 10:34 UTC'
    );
  });
});

describe('getCheckStatus', () => {
  it('reports an update in progress', () => {
    expect(getCheckStatus(versionInfo(null), true)).toBe(Messages.inProgress);
  });

  it('prefers the in progress message over the last check', () => {
    expect(
      getCheckStatus(versionInfo('2024-07-30T10:34:05.886739003Z'), true)
    ).toBe(Messages.inProgress);
  });

  it('reports the last check date', () => {
    expect(
      getCheckStatus(versionInfo('2024-07-30T10:34:05.886739003Z'), false)
    ).toBe(Messages.checkedOn('July 30, 2024, 10:34 UTC'));
  });

  it('reports nothing when no check has run', () => {
    expect(getCheckStatus(versionInfo(null), false)).toBeNull();
  });

  it('reports nothing without version info', () => {
    expect(getCheckStatus(undefined, false)).toBeNull();
  });
});
