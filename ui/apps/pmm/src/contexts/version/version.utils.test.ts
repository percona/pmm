import {
  AUTO_RELOAD_COOLDOWN_MS,
  canAutoReload,
  getServerBuildId,
  hasBuildChanged,
  recordAutoReload,
} from './version.utils';

const NOW = new Date('2026-08-19T10:00:00Z').getTime();

describe('getServerBuildId', () => {
  it('prefers the managed commit', () => {
    expect(
      getServerBuildId({
        version: '3.10.0',
        server: { version: '3.10.0', fullVersion: '3.10.0-1' },
        managed: { version: '3.10.0', fullVersion: 'abc123' },
      })
    ).toBe('abc123');
  });

  it('falls back to the server full version', () => {
    expect(
      getServerBuildId({
        version: '3.10.0',
        server: { version: '3.10.0', fullVersion: '3.10.0-1' },
        managed: { version: '3.10.0', fullVersion: '' },
      })
    ).toBe('3.10.0-1');
  });

  it('falls back to the user-visible version', () => {
    expect(getServerBuildId({ version: '3.10.0' })).toBe('3.10.0');
  });

  it('is empty without data', () => {
    expect(getServerBuildId()).toBe('');
  });
});

describe('hasBuildChanged', () => {
  it('reports a different build', () => {
    expect(hasBuildChanged('abc123', 'def456')).toBe(true);
  });

  it('ignores an unchanged build', () => {
    expect(hasBuildChanged('abc123', 'abc123')).toBe(false);
  });

  it('ignores a missing baseline or reading', () => {
    expect(hasBuildChanged('', 'def456')).toBe(false);
    expect(hasBuildChanged('abc123', '')).toBe(false);
  });
});

describe('auto reload cooldown', () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it('allows a reload when none was recorded', () => {
    expect(canAutoReload(NOW)).toBe(true);
  });

  it('blocks a second reload within the cooldown', () => {
    recordAutoReload(NOW);

    expect(canAutoReload(NOW + AUTO_RELOAD_COOLDOWN_MS - 1)).toBe(false);
  });

  it('allows a reload once the cooldown elapsed', () => {
    recordAutoReload(NOW);

    expect(canAutoReload(NOW + AUTO_RELOAD_COOLDOWN_MS)).toBe(true);
  });

  it('reports that it recorded the reload', () => {
    expect(recordAutoReload(NOW)).toBe(true);
  });

  it('reports a refusal instead of throwing when storage is unavailable', () => {
    const getItem = vi
      .spyOn(Storage.prototype, 'getItem')
      .mockImplementation(() => {
        throw new Error('storage disabled');
      });
    const setItem = vi
      .spyOn(Storage.prototype, 'setItem')
      .mockImplementation(() => {
        throw new Error('storage disabled');
      });

    // the caller has to decline the reload: with nothing written down the
    // allowance cannot be enforced on the next poll
    expect(recordAutoReload(NOW)).toBe(false);
    expect(() => canAutoReload(NOW)).not.toThrow();

    getItem.mockRestore();
    setItem.mockRestore();
  });
});
