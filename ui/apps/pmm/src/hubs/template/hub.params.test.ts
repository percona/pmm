import {
  DEFAULT_HUB_PARAMS,
  environmentOptions,
  matchesEnvironment,
  HUB_REFRESHES,
  parseHubParams,
  refreshInterval,
  resolveHubCapabilities,
  resolveHubOperations,
  resolveHubRole,
  writeHubSearch,
} from './hub.params';
import { HUB_NO_ENVIRONMENT } from './hub.types';

describe('parseHubParams', () => {
  it('defaults to the best case with Operations on, every environment, as signed in', () => {
    expect(parseHubParams('')).toEqual(DEFAULT_HUB_PARAMS);
    expect(DEFAULT_HUB_PARAMS).toMatchObject({
      scenario: 'healthy',
      ops: 'on',
      role: 'live',
      refresh: '30s',
      freshness: 'live',
    });
  });

  it('reads Live data and As configured from the URL', () => {
    expect(parseHubParams('?hubState=live&hubOps=live')).toMatchObject({
      scenario: 'live',
      ops: 'live',
    });
  });

  it('reads every param', () => {
    expect(
      parseHubParams(
        '?environment=prod&refresh=1h&hubState=unhealthy&hubOps=off&hubRole=viewer&hubFreshness=stale'
      )
    ).toEqual({
      environment: 'prod',
      refresh: '1h',
      scenario: 'unhealthy',
      ops: 'off',
      role: 'viewer',
      freshness: 'stale',
    });
  });

  it('falls back to the defaults for unknown values', () => {
    expect(
      parseHubParams(
        new URLSearchParams(
          'hubState=broken&hubOps=maybe&hubRole=root&refresh=2s&hubFreshness=off'
        )
      )
    ).toEqual(DEFAULT_HUB_PARAMS);
  });

  it('reads an empty environment as every environment', () => {
    expect(parseHubParams('environment=').environment).toBeNull();
  });

  it('keeps the no-environment value', () => {
    expect(
      parseHubParams(`environment=${HUB_NO_ENVIRONMENT}`).environment
    ).toBe(HUB_NO_ENVIRONMENT);
  });
});

describe('writeHubSearch', () => {
  it('leaves defaults out of the URL', () => {
    expect(writeHubSearch('?hubState=unhealthy', { scenario: 'healthy' })).toBe(
      ''
    );
  });

  it('writes Live data and As configured, which are not the defaults', () => {
    expect(writeHubSearch('', { scenario: 'live', ops: 'live' })).toBe(
      'hubState=live&hubOps=live'
    );
  });

  it('keeps unrelated params byte for byte', () => {
    expect(
      writeHubSearch('?from=now-1h&q=a%20b+c&tag=x&tag=y', {
        scenario: 'unhealthy',
      })
    ).toBe('from=now-1h&q=a%20b+c&tag=x&tag=y&hubState=unhealthy');
  });

  it('replaces the hub params it is given and keeps the others', () => {
    expect(
      writeHubSearch('environment=prod&hubState=unhealthy&hubRole=viewer', {
        environment: 'staging',
      })
    ).toBe('environment=staging&hubState=unhealthy&hubRole=viewer');
  });

  it('removes the environment when every environment is selected', () => {
    expect(writeHubSearch('environment=prod&x=1', { environment: null })).toBe(
      'x=1'
    );
  });

  it('encodes values', () => {
    expect(writeHubSearch('', { environment: 'prod eu' })).toBe(
      'environment=prod+eu'
    );
  });
});

describe('writeHubSearch with every param', () => {
  it('writes only non-default hub params in a stable order', () => {
    expect(
      writeHubSearch('', {
        environment: 'prod',
        refresh: 'off',
        freshness: 'stale',
        scenario: 'unhealthy',
        ops: 'off',
        role: 'live',
      })
    ).toBe(
      'environment=prod&refresh=off&hubState=unhealthy&hubOps=off&hubFreshness=stale'
    );
    expect(writeHubSearch('', DEFAULT_HUB_PARAMS)).toBe('');
  });
});

describe('refreshInterval', () => {
  it('offers Grafana’s interval set, Off first', () => {
    expect(HUB_REFRESHES).toEqual([
      'off',
      '5s',
      '10s',
      '30s',
      '1m',
      '5m',
      '15m',
      '30m',
      '1h',
      '2h',
      '1d',
    ]);
  });

  it('reads each choice in ms, and off as no interval', () => {
    expect(refreshInterval('5s')).toBe(5_000);
    expect(refreshInterval('30s')).toBe(30_000);
    expect(refreshInterval('15m')).toBe(15 * 60_000);
    expect(refreshInterval('1d')).toBe(24 * 60 * 60_000);
    expect(refreshInterval('off')).toBe(false);
  });
});

describe('resolveHubRole', () => {
  it('follows the signed-in user by default', () => {
    expect(resolveHubRole(DEFAULT_HUB_PARAMS, true)).toBe('admin');
    expect(resolveHubRole(DEFAULT_HUB_PARAMS, false)).toBe('viewer');
  });

  it('lets hubRole=viewer lower an admin, live or not', () => {
    expect(resolveHubRole({ scenario: 'live', role: 'viewer' }, true)).toBe(
      'viewer'
    );
    expect(
      resolveHubRole({ scenario: 'unhealthy', role: 'viewer' }, true)
    ).toBe('viewer');
  });

  it('ignores hubRole=admin on live data', () => {
    expect(resolveHubRole({ scenario: 'live', role: 'admin' }, false)).toBe(
      'viewer'
    );
  });

  it('honours hubRole=admin on sample data', () => {
    expect(resolveHubRole({ scenario: 'healthy', role: 'admin' }, false)).toBe(
      'admin'
    );
  });
});

describe('resolveHubOperations', () => {
  it('is on for an admin by default, on sample data', () => {
    expect(resolveHubOperations(DEFAULT_HUB_PARAMS, 'admin', false)).toBe(true);
  });

  it('follows the setting as configured', () => {
    const asConfigured = { scenario: 'healthy', ops: 'live' } as const;

    expect(resolveHubOperations(asConfigured, 'admin', true)).toBe(true);
    expect(resolveHubOperations(asConfigured, 'admin', false)).toBe(false);
  });

  it('lets hubOps=off lower it, live or not', () => {
    expect(
      resolveHubOperations({ scenario: 'live', ops: 'off' }, 'admin', true)
    ).toBe(false);
    expect(
      resolveHubOperations({ scenario: 'unhealthy', ops: 'off' }, 'admin', true)
    ).toBe(false);
  });

  it('ignores hubOps=on on live data', () => {
    expect(
      resolveHubOperations({ scenario: 'live', ops: 'on' }, 'admin', false)
    ).toBe(false);
  });

  it('honours hubOps=on on sample data', () => {
    expect(
      resolveHubOperations({ scenario: 'empty', ops: 'on' }, 'admin', false)
    ).toBe(true);
  });

  it('never gives a viewer Operations', () => {
    expect(
      resolveHubOperations({ scenario: 'unhealthy', ops: 'on' }, 'viewer', true)
    ).toBe(false);
  });
});

describe('resolveHubCapabilities', () => {
  const base = {
    operations: true,
    backupManagementEnabled: true,
    extensionsEnabled: true,
    signedIn: true,
  };

  it('gives an admin with Operations every action', () => {
    expect(resolveHubCapabilities({ ...base, role: 'admin' })).toEqual({
      addService: true,
      install: true,
      operations: true,
      setUpBackup: true,
      extensions: true,
    });
  });

  it('gives an admin without Operations no install', () => {
    expect(
      resolveHubCapabilities({ ...base, role: 'admin', operations: false })
    ).toMatchObject({ addService: true, install: false, operations: false });
  });

  it('gives a viewer no actions but keeps PMM Extensions', () => {
    expect(
      resolveHubCapabilities({ ...base, role: 'viewer', operations: false })
    ).toEqual({
      addService: false,
      install: false,
      operations: false,
      setUpBackup: false,
      extensions: true,
    });
  });

  it('offers backup setup only with backup management on', () => {
    expect(
      resolveHubCapabilities({
        ...base,
        role: 'admin',
        backupManagementEnabled: false,
      }).setUpBackup
    ).toBe(false);
  });

  it('needs a signed-in user for PMM Extensions', () => {
    expect(
      resolveHubCapabilities({ ...base, role: 'viewer', signedIn: false })
        .extensions
    ).toBe(false);
  });
});

describe('matchesEnvironment', () => {
  it('passes everything without a filter', () => {
    expect(matchesEnvironment('prod', null)).toBe(true);
    expect(matchesEnvironment(undefined, null)).toBe(true);
  });

  it('matches one environment', () => {
    expect(matchesEnvironment('prod', 'prod')).toBe(true);
    expect(matchesEnvironment('staging', 'prod')).toBe(false);
  });

  it('matches services without an environment for No environment', () => {
    expect(matchesEnvironment(undefined, HUB_NO_ENVIRONMENT)).toBe(true);
    expect(matchesEnvironment('', HUB_NO_ENVIRONMENT)).toBe(true);
    expect(matchesEnvironment('prod', HUB_NO_ENVIRONMENT)).toBe(false);
  });
});

describe('environmentOptions', () => {
  it('lists each environment once, alphabetically', () => {
    expect(
      environmentOptions(['staging', 'prod', 'prod'], 'No environment')
    ).toEqual([
      { value: 'prod', label: 'prod' },
      { value: 'staging', label: 'staging' },
    ]);
  });

  it('adds No environment last when a service has none', () => {
    expect(environmentOptions(['prod', undefined], 'No environment')).toEqual([
      { value: 'prod', label: 'prod' },
      { value: HUB_NO_ENVIRONMENT, label: 'No environment' },
    ]);
  });

  it('is empty for no services', () => {
    expect(environmentOptions([], 'No environment')).toEqual([]);
  });

  it('keeps a selected environment that no service has', () => {
    expect(environmentOptions(['prod'], 'No environment', 'staging')).toEqual([
      { value: 'prod', label: 'prod' },
      { value: 'staging', label: 'staging' },
    ]);
    expect(
      environmentOptions(['prod'], 'No environment', HUB_NO_ENVIRONMENT)
    ).toEqual([
      { value: 'prod', label: 'prod' },
      { value: HUB_NO_ENVIRONMENT, label: 'No environment' },
    ]);
  });
});
