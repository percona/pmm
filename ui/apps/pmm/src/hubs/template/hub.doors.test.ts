import { buildTableUrlParamKeys } from '@percona/peak-ui';
import {
  ALERTS_SERVICE_FILTER_KEY,
  alertsLink,
  grafanaDashboard,
  INCIDENT_LEAD_MS,
  listEnvironment,
  qanLink,
  realtimeLink,
  withIncidentFrame,
  withQuery,
} from './hub.doors';
import { HUB_NO_ENVIRONMENT } from './hub.types';

describe('withQuery', () => {
  it('appends pairs and skips empty values', () => {
    expect(withQuery('/x', { a: '1', b: undefined, c: null, d: '' })).toBe(
      '/x?a=1'
    );
  });

  it('returns the path alone without values', () => {
    expect(withQuery('/x', {})).toBe('/x');
  });

  it('keeps a list readable, commas as they are', () => {
    expect(withQuery('/x', { clusters: 'cart-eu,users-eu' })).toBe(
      '/x?clusters=cart-eu,users-eu'
    );
  });

  it('extends an existing query', () => {
    expect(withQuery('/x?tab=clusters', { cluster: 'om-1' })).toBe(
      '/x?tab=clusters&cluster=om-1'
    );
  });

  it('repeats array values', () => {
    expect(withQuery('/x', { id: ['a', 'b'] })).toBe('/x?id=a&id=b');
  });

  it('encodes spaces as %20 and quotes, ampersands and slashes', () => {
    expect(withQuery('/x', { name: 'my "rs" & co/1' })).toBe(
      '/x?name=my%20%22rs%22%20%26%20co%2F1'
    );
  });
});

describe('grafanaDashboard', () => {
  it('prefixes variables with var- and keeps other params', () => {
    expect(
      grafanaDashboard(
        'mongodb-replicaset-summary/mongodb-replset-summary',
        { environment: 'prod', cluster: undefined, rs_nm: 'billing' },
        { from: 'now-3h' }
      )
    ).toBe(
      '/graph/d/mongodb-replicaset-summary/mongodb-replset-summary?var-environment=prod&var-rs_nm=billing&from=now-3h'
    );
  });
});

describe('listEnvironment', () => {
  it('carries the filter only when one real environment is selected', () => {
    expect(listEnvironment('prod')).toBe('prod');
    expect(listEnvironment(null)).toBeUndefined();
    expect(listEnvironment(HUB_NO_ENVIRONMENT)).toBeUndefined();
  });
});

describe('withIncidentFrame', () => {
  it('opens 30 minutes before the incident and runs to now', () => {
    const since = '2026-10-19T12:00:00.000Z';

    expect(withIncidentFrame('/graph/d/x?var-a=1', since)).toBe(
      `/graph/d/x?var-a=1&from=${Date.parse(since) - INCIDENT_LEAD_MS}&to=now`
    );
  });

  it('leaves the door alone without a valid start', () => {
    expect(withIncidentFrame('/graph/d/x', undefined)).toBe('/graph/d/x');
    expect(withIncidentFrame('/graph/d/x', 'not a date')).toBe('/graph/d/x');
  });
});

describe('qanLink', () => {
  it('opens Query analytics for a service type and environment', () => {
    expect(qanLink({ serviceType: 'mongodb', environment: 'prod' })).toBe(
      '/graph/d/pmm-qan/pmm-query-analytics?var-service_type=mongodb&var-environment=prod'
    );
  });

  it('opens one query with its details over the last hour', () => {
    expect(qanLink({ serviceType: 'mongodb', queryId: 'Q1' })).toBe(
      '/graph/d/pmm-qan/pmm-query-analytics?var-service_type=mongodb&from=now-1h&to=now&filter_by=Q1&query_selected=true&details_tab=details'
    );
  });
});

describe('alertsLink', () => {
  it('uses the key the Alerts table reads its column filters from', () => {
    expect(ALERTS_SERVICE_FILTER_KEY).toBe(
      `${buildTableUrlParamKeys().filterPrefix}serviceName`
    );
  });

  it('filters to one service', () => {
    expect(alertsLink('orders-shard2-m2')).toBe(
      `/alerting/status?${ALERTS_SERVICE_FILTER_KEY}=orders-shard2-m2`
    );
  });

  it('opens every alert without a service', () => {
    expect(alertsLink()).toBe('/alerting/status');
  });
});

describe('realtimeLink', () => {
  it('opens the overview for the given services', () => {
    expect(realtimeLink(['a', 'b'])).toBe(
      '/rta/overview?serviceIds=a&serviceIds=b'
    );
  });

  it('opens Real-time queries without services', () => {
    expect(realtimeLink()).toBe('/rta');
  });
});
