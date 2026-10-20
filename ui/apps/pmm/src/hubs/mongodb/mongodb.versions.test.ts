import {
  compareVersions,
  groupByVersion,
  lowestVersion,
} from './mongodb.versions';

describe('compareVersions', () => {
  it('orders by release, then by Percona build', () => {
    expect(
      [
        '7.0.30-16',
        '7.0.24-13',
        '8.0.12-4',
        '7.0.24-9',
        '6.0.25-20',
        '7.0.4-2',
      ].sort(compareVersions)
    ).toEqual([
      '6.0.25-20',
      '7.0.4-2',
      '7.0.24-9',
      '7.0.24-13',
      '7.0.30-16',
      '8.0.12-4',
    ]);
  });

  it('puts an upstream release below its Percona builds', () => {
    expect(compareVersions('7.0.15', '7.0.15-9')).toBeLessThan(0);
    expect(compareVersions('7.0.15', '7.0.15')).toBe(0);
  });

  it('falls back to text order for what is no version', () => {
    expect(compareVersions('unknown', 'zeta')).toBeLessThan(0);
  });
});

describe('lowestVersion', () => {
  it('is the version a group is still exposed on', () => {
    expect(lowestVersion(['7.0.30-16', '7.0.24-13', '7.0.30-16'])).toBe(
      '7.0.24-13'
    );
    expect(lowestVersion([])).toBeUndefined();
  });
});

describe('groupByVersion', () => {
  it('groups members by version, lowest first, and skips those without one', () => {
    expect(
      groupByVersion([
        { name: 'c-m3', version: '7.0.30-16' },
        { name: 'c-m1', version: '7.0.24-13' },
        { name: 'c-m2', version: '7.0.30-16' },
        { name: 'c-m4' },
      ])
    ).toEqual([
      { version: '7.0.24-13', members: ['c-m1'] },
      { version: '7.0.30-16', members: ['c-m2', 'c-m3'] },
    ]);
  });
});
