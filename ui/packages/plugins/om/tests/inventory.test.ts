/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

import { describe, expect, it } from 'vitest';
import {
  ageSeconds,
  databaseState,
  isBoundedPeriod,
  isFailing,
  isRunPeriod,
  joinServiceInventory,
  periodSince,
  repoReachability,
  toHostRows,
} from '../src/inventory';
import type {
  OmInventoryFreshness,
  OmInventoryHost,
  OmInventoryService,
  OmServiceRow,
  OmUnregisteredMongod,
} from '../src/types';

const freshness = (
  overrides: Partial<OmInventoryFreshness> = {}
): OmInventoryFreshness => ({
  first_seen_at: '2026-08-17T09:00:00Z',
  last_attempt_at: '2026-08-18T09:00:00Z',
  last_success_at: '2026-08-18T09:00:00Z',
  failing_since: null,
  consecutive_failures: 0,
  last_error: null,
  ...overrides,
});

const mongod = (
  overrides: Partial<OmUnregisteredMongod> = {}
): OmUnregisteredMongod => ({
  port: 27018,
  config_path: '/etc/mongod-node.conf',
  argv: '/usr/bin/mongod --config /etc/mongod-node.conf',
  program: 'mongod',
  pid: 10,
  ...overrides,
});

const service = (
  overrides: Partial<OmInventoryService> = {}
): OmInventoryService => ({
  service_id: 'svc-1',
  node_id: 'node-1',
  name: 'db00',
  port: 27017,
  role: null,
  installed_version: '7.0.39-21',
  running_version: '7.0.39-21',
  config_path: '/etc/mongod-node.conf',
  argv: '/usr/bin/mongod',
  probe_status: 'ok',
  server_running: true,
  uptime_seconds: 100,
  replication_set: 'rs0',
  observed: {},
  freshness: freshness(),
  ...overrides,
});

const host = (overrides: Partial<OmInventoryHost> = {}): OmInventoryHost => ({
  node_id: 'node-1',
  name: 'db00',
  address: '10.0.0.1',
  executor_host: 'db00',
  os: 'Ubuntu 24.04.3 LTS',
  kernel: '6.17.0-35-generic',
  executor: {
    registered: true,
    reachable: true,
    driver_healthy: true,
    detail: null,
  },
  unregistered_mongods: [],
  observed: {},
  freshness: freshness(),
  services: [],
  pmm_agent_connected: true,
  automation_eligible: true,
  automation_blocked_reasons: [],
  ...overrides,
});

const snapshotService = (overrides: Partial<OmServiceRow> = {}): OmServiceRow =>
  ({
    service_id: 'svc-1',
    service_name: 'db00',
    host: 'node00',
    version: '7.0.39-21',
    env_name: 'prod',
    cluster_name: 'rs0',
    ...overrides,
  }) as OmServiceRow;

describe('databaseState', () => {
  it('reports a host with a registered service as having one', () => {
    expect(databaseState(host({ services: [service()] }))).toBe('has_service');
  });

  it('reports a host with nothing on it as installable', () => {
    expect(databaseState(host())).toBe('installable');
  });

  /**
   * The case the whole three-state column exists for.
   *
   * PMM registers no service for an arbiter, because it cannot authenticate against
   * one, so the host looks identical to a bare pmm-client in PMM's own inventory:
   * same node type, same agents, no services. Calling it installable would invite
   * someone to put a database on a port already in use.
   */
  it('does not call a host with an unregistered mongod installable', () => {
    const arbiter = host({ unregistered_mongods: [mongod()] });

    expect(databaseState(arbiter)).toBe('unregistered_only');
  });

  it('prefers a registered service when a host has both', () => {
    const both = host({
      services: [service()],
      unregistered_mongods: [mongod({ port: 27018 })],
    });

    // The arbiter still shows in its own column; the state answers "is there a
    // database PMM knows about", and here there is.
    expect(databaseState(both)).toBe('has_service');
  });
});

describe('repoReachability', () => {
  it('reads the record the probe stored', () => {
    const withRepo = host({
      observed: {
        repo: {
          url: 'https://repo.percona.com/percona/yum/PERCONA-PACKAGING-KEY',
          reachable: true,
          status_code: 200,
          latency_ms: 141,
          proxy: null,
          error: null,
        },
      },
    });

    expect(repoReachability(withRepo)).toEqual({
      url: 'https://repo.percona.com/percona/yum/PERCONA-PACKAGING-KEY',
      reachable: true,
      status_code: 200,
      latency_ms: 141,
      proxy: null,
      error: null,
    });
  });

  it('carries the proxy and the reason when the fetch failed', () => {
    const blocked = host({
      observed: {
        repo: {
          reachable: false,
          proxy: 'http://proxy.internal:3128',
          error: 'HTTP 403: Forbidden',
        },
      },
    });

    const repo = repoReachability(blocked);

    expect(repo?.reachable).toBe(false);
    expect(repo?.proxy).toBe('http://proxy.internal:3128');
    expect(repo?.error).toBe('HTTP 403: Forbidden');
  });

  /**
   * `observed` is an untyped bag by design, so nothing upstream promises this key
   * exists or that it is an object. A host probed before the check was added has no
   * `repo` at all, and that must read as "not known" rather than "unreachable" - the
   * difference between a host nobody has asked and a host that answered no.
   */
  it('returns null when the host has never reported one', () => {
    expect(repoReachability(host())).toBeNull();
    expect(repoReachability(host({ observed: { repo: null } }))).toBeNull();
    expect(repoReachability(host({ observed: { repo: 'yes' } }))).toBeNull();
    expect(repoReachability(host({ observed: { repo: [] } }))).toBeNull();
  });

  it('treats a missing reachable flag as not reachable', () => {
    const partial = host({ observed: { repo: { url: 'https://x' } } });

    expect(repoReachability(partial)?.reachable).toBe(false);
  });
});

describe('toHostRows', () => {
  it('derives the state, the count and the repository per host', () => {
    const rows = toHostRows([
      host({ services: [service(), service({ service_id: 'svc-2' })] }),
      host({ node_id: 'node-2', name: 'pmm-client-node00' }),
    ]);

    expect(rows[0].database_state).toBe('has_service');
    expect(rows[0].service_count).toBe(2);
    expect(rows[1].database_state).toBe('installable');
    expect(rows[1].repo).toBeNull();
  });

  it('returns nothing while the estate is still loading', () => {
    expect(toHostRows(undefined)).toEqual([]);
  });
});

describe('joinServiceInventory', () => {
  it('joins on the service id with no matching rule', () => {
    const rows = joinServiceInventory(
      [snapshotService()],
      [service({ service_id: 'svc-1', installed_version: '7.0.40-22' })]
    );

    expect(rows[0].inventory?.installed_version).toBe('7.0.40-22');
  });

  /**
   * Rows come from the snapshot, so a service PMM has registered since the last sweep
   * still appears - with its probe columns unavailable rather than the row missing.
   * Dropping it would make the Services page quietly narrower than PMM's own
   * inventory, which is the opposite of what joining a second source is for.
   */
  it('keeps a service the estate has never probed', () => {
    const rows = joinServiceInventory([snapshotService()], []);

    expect(rows).toHaveLength(1);
    expect(rows[0].inventory).toBeNull();
  });

  /**
   * The mirror image: the estate keeps rows for services PMM no longer has, because a
   * `setup --force` re-registration mints new ids and nothing prunes the old ones.
   * Those belong behind the Hosts page's delete action, not silently added to a table
   * of what PMM currently monitors.
   */
  it('does not invent a row for an estate entry PMM no longer has', () => {
    const rows = joinServiceInventory(
      [snapshotService()],
      [service({ service_id: 'svc-1' }), service({ service_id: 'svc-gone' })]
    );

    expect(rows.map((row) => row.service_id)).toEqual(['svc-1']);
  });

  it('survives an estate that has not loaded yet', () => {
    expect(joinServiceInventory([snapshotService()], undefined)).toHaveLength(
      1
    );
  });
});

describe('ageSeconds', () => {
  const now = Date.parse('2026-08-18T12:00:00Z');

  it('measures back from now', () => {
    expect(ageSeconds('2026-08-18T11:59:00Z', now)).toBe(60);
  });

  it('reports null for a row that has never answered', () => {
    expect(ageSeconds(null, now)).toBeNull();
    expect(ageSeconds(undefined, now)).toBeNull();
    expect(ageSeconds('not a date', now)).toBeNull();
  });

  /**
   * A host clock a few seconds ahead of the browser's is ordinary. Reporting "in -4
   * seconds" would read as a bug in this page rather than as the clock skew it is.
   */
  it('clamps a timestamp in the future to zero', () => {
    expect(ageSeconds('2026-08-18T12:00:04Z', now)).toBe(0);
  });
});

describe('isFailing', () => {
  it('is true while a row has not succeeded since its first failure', () => {
    const failing = host({
      freshness: freshness({
        failing_since: '2026-08-15T09:00:00Z',
        consecutive_failures: 37,
        last_success_at: null,
      }),
    });

    expect(isFailing(failing)).toBe(true);
  });

  it('is false for a healthy row', () => {
    expect(isFailing(host())).toBe(false);
  });

  /**
   * Reads `failing_since` rather than the counter. They are reset together, so the
   * counter looks like an equivalent test - but only the timestamp answers "since
   * when", and a row that has failed exactly once is still failing.
   */
  it('is true after a single failure, before any count builds up', () => {
    const once = host({
      freshness: freshness({
        failing_since: '2026-08-18T11:00:00Z',
        consecutive_failures: 1,
      }),
    });

    expect(isFailing(once)).toBe(true);
  });
});

describe('periodSince', () => {
  const now = new Date('2026-08-25T12:34:56.000Z');

  it('returns fifteen minutes back', () => {
    expect(periodSince('15m', now)).toBe('2026-08-25T12:19:56.000Z');
  });

  it('returns thirty minutes back', () => {
    expect(periodSince('30m', now)).toBe('2026-08-25T12:04:56.000Z');
  });

  it('returns one hour back', () => {
    expect(periodSince('1h', now)).toBe('2026-08-25T11:34:56.000Z');
  });

  it('returns four hours back', () => {
    expect(periodSince('4h', now)).toBe('2026-08-25T08:34:56.000Z');
  });

  it('returns eight hours back', () => {
    expect(periodSince('8h', now)).toBe('2026-08-25T04:34:56.000Z');
  });

  it("returns today's local midnight, not a rolling twenty-four hours", () => {
    const midnight = new Date(now);
    midnight.setHours(0, 0, 0, 0);
    expect(periodSince('today', now)).toBe(midnight.toISOString());
  });

  it('returns seven days back for last week', () => {
    expect(periodSince('week', now)).toBe('2026-08-18T12:34:56.000Z');
  });

  it('returns thirty days back for last month', () => {
    expect(periodSince('month', now)).toBe('2026-07-26T12:34:56.000Z');
  });

  it('omits the bound for all', () => {
    expect(periodSince('all', now)).toBeUndefined();
  });

  it('recognises the configured windows and nothing else', () => {
    expect(isRunPeriod('week')).toBe(true);
    expect(isRunPeriod('15m')).toBe(true);
    expect(isRunPeriod('today')).toBe(true);
    expect(isRunPeriod('year')).toBe(false);
    expect(isRunPeriod(null)).toBe(false);
  });

  it('is bounded for every period except all', () => {
    expect(isBoundedPeriod('15m')).toBe(true);
    expect(isBoundedPeriod('today')).toBe(true);
    expect(isBoundedPeriod('week')).toBe(true);
    expect(isBoundedPeriod('all')).toBe(false);
  });
});
