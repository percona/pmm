import {
  DEFAULT_HUB_PARAMS,
  environmentOptions,
  matchesEnvironment,
  parseHubParams,
  resolveHubCapabilities,
  resolveHubOperations,
  resolveHubRole,
  serializeHubParams,
  writeHubSearch,
} from './hub.params';
import { HUB_NO_ENVIRONMENT } from './hub.types';

describe('parseHubParams', () => {
  it('defaults to live data, every environment, as configured and as signed in', () => {
    expect(parseHubParams('')).toEqual(DEFAULT_HUB_PARAMS);
  });

  it('reads every param', () => {
    expect(
      parseHubParams(
        '?environment=prod&hubState=wrong&hubOps=off&hubRole=viewer'
      )
    ).toEqual({
      environment: 'prod',
      scenario: 'wrong',
      ops: 'off',
      role: 'viewer',
    });
  });

  it('falls back to live for unknown values', () => {
    expect(
      parseHubParams(
        new URLSearchParams('hubState=broken&hubOps=maybe&hubRole=root')
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
    expect(writeHubSearch('?hubState=wrong', { scenario: 'live' })).toBe('');
  });

  it('keeps unrelated params byte for byte', () => {
    expect(
      writeHubSearch('?from=now-1h&q=a%20b+c&tag=x&tag=y', {
        scenario: 'healthy',
      })
    ).toBe('from=now-1h&q=a%20b+c&tag=x&tag=y&hubState=healthy');
  });

  it('replaces the hub params it is given and keeps the others', () => {
    expect(
      writeHubSearch('environment=prod&hubState=wrong&hubRole=viewer', {
        environment: 'staging',
      })
    ).toBe('environment=staging&hubState=wrong&hubRole=viewer');
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

describe('serializeHubParams', () => {
  it('writes only non-default hub params in a stable order', () => {
    expect(
      serializeHubParams({
        environment: 'prod',
        scenario: 'wrong',
        ops: 'on',
        role: 'live',
      })
    ).toBe('environment=prod&hubState=wrong&hubOps=on');
    expect(serializeHubParams(DEFAULT_HUB_PARAMS)).toBe('');
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
    expect(resolveHubRole({ scenario: 'wrong', role: 'viewer' }, true)).toBe(
      'viewer'
    );
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
  it('follows the setting by default', () => {
    expect(resolveHubOperations(DEFAULT_HUB_PARAMS, 'admin', true)).toBe(true);
    expect(resolveHubOperations(DEFAULT_HUB_PARAMS, 'admin', false)).toBe(
      false
    );
  });

  it('lets hubOps=off lower it, live or not', () => {
    expect(
      resolveHubOperations({ scenario: 'live', ops: 'off' }, 'admin', true)
    ).toBe(false);
    expect(
      resolveHubOperations({ scenario: 'wrong', ops: 'off' }, 'admin', true)
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
      resolveHubOperations({ scenario: 'wrong', ops: 'on' }, 'viewer', true)
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
