import {
  alertDoor,
  automationsDoor,
  backupStatus,
  clusterQan,
  clusterSummary,
  instanceSummary,
  lagDoor,
  listDashboard,
  manageDoor,
  MONGO_PATHS,
  oplogDetails,
  queryQan,
  replsetSummary,
  routerSummary,
  sectionAnchor,
} from './mongodb.doors';
import { DEFAULT_HUB_PARAMS } from 'hubs/template/hub.params';
import { HUB_NO_ENVIRONMENT } from 'hubs/template/hub.types';

const ORDERS = { environment: 'prod', cluster: 'orders' };

describe('entity doors carry the entity’s own environment and cluster', () => {
  it('sharded cluster summary', () => {
    expect(clusterSummary(ORDERS)).toBe(
      '/graph/d/mongodb-cluster-summary/mongodb-sharded-cluster-summary?var-environment=prod&var-cluster=orders'
    );
  });

  it('ReplSet summary', () => {
    expect(replsetSummary(ORDERS, 'orders-shard1')).toBe(
      '/graph/d/mongodb-replicaset-summary/mongodb-replset-summary?var-environment=prod&var-cluster=orders&var-rs_nm=orders-shard1'
    );
  });

  it('Router summary, which needs an environment', () => {
    expect(routerSummary(ORDERS)).toBe(
      '/graph/d/mongodb-router-summary/mongodb-router-summary?var-environment=prod&var-cluster=orders'
    );
  });

  it('Instance summary', () => {
    expect(instanceSummary(ORDERS, 'orders-shard1-m1')).toBe(
      '/graph/d/mongodb-instance-summary/mongodb-instance-summary?var-environment=prod&var-cluster=orders&var-service_name=orders-shard1-m1'
    );
  });

  it('Oplog details on one member', () => {
    expect(oplogDetails(ORDERS, 'orders-shard2', 'orders-shard2-m1')).toBe(
      '/graph/d/mongodb-oplog-details/mongodb-oplog-details?var-environment=prod&var-cluster=orders&var-replset=orders-shard2&var-service_name=orders-shard2-m1'
    );
  });

  it('Backup status and Query analytics for the cluster', () => {
    expect(backupStatus(ORDERS)).toBe(
      '/graph/d/mongodb-backup-details/mongodb-backup-details?var-environment=prod&var-cluster=orders'
    );
    expect(clusterQan(ORDERS)).toBe(
      '/graph/d/pmm-qan/pmm-query-analytics?var-service_type=mongodb&var-environment=prod&var-cluster=orders'
    );
  });

  it('omits the environment of a service without one', () => {
    expect(instanceSummary({ cluster: 'c' }, 'm')).toBe(
      '/graph/d/mongodb-instance-summary/mongodb-instance-summary?var-cluster=c&var-service_name=m'
    );
  });

  it('encodes names with spaces and quotes', () => {
    expect(
      instanceSummary({ environment: 'prod eu', cluster: 'a"b' }, 'x y')
    ).toBe(
      '/graph/d/mongodb-instance-summary/mongodb-instance-summary?var-environment=prod%20eu&var-cluster=a%22b&var-service_name=x%20y'
    );
  });

  it('opens lag over the last 3 hours with hidden context', () => {
    expect(lagDoor(ORDERS, 'orders-shard1', 'orders-shard1-m3')).toEqual({
      label: 'orders-shard1-m3',
      to: '/graph/d/mongodb-replicaset-summary/mongodb-replset-summary?var-environment=prod&var-cluster=orders&var-rs_nm=orders-shard1&from=now-3h&to=now',
      context: 'replication lag of orders-shard1-m3',
    });
  });
});

describe('list doors carry the filter only when one environment is selected', () => {
  it('adds var-environment for a selected environment', () => {
    expect(listDashboard('instancesOverview', 'prod')).toBe(
      '/graph/d/mongodb-instance-overview/mongodb-instances-overview?var-environment=prod'
    );
  });

  it('adds nothing for all environments or No environment', () => {
    expect(listDashboard('instancesOverview', null)).toBe(
      '/graph/d/mongodb-instance-overview/mongodb-instances-overview'
    );
    expect(listDashboard('instancesOverview', HUB_NO_ENVIRONMENT)).toBe(
      '/graph/d/mongodb-instance-overview/mongodb-instances-overview'
    );
  });

  it('opens one query in Query analytics', () => {
    expect(queryQan('prod', 'ABC123')).toBe(
      '/graph/d/pmm-qan/pmm-query-analytics?var-service_type=mongodb&var-environment=prod&from=now-1h&to=now&filter_by=ABC123&query_selected=true&details_tab=details'
    );
  });
});

describe('Operations doors carry Preview', () => {
  it('Manage in Operations focuses the cluster in Fleet', () => {
    expect(manageDoor('om-orders')).toEqual({
      label: 'Manage in Operations',
      to: '/operations?tab=clusters&cluster=om-orders',
      preview: 'developer',
    });
    expect(manageDoor().to).toBe('/operations?tab=clusters');
  });

  it('a failed run opens in Automations, expanded', () => {
    expect(automationsDoor('scan', 'run 1')).toEqual({
      label: 'Open Automations',
      to: '/operations/automations?tab=scans&expand=run%201',
      preview: 'developer',
    });
  });

  it('Install MongoDB opens Nodes on the unmonitored ones', () => {
    expect(MONGO_PATHS.install).toBe('/operations/nodes?filter=unmonitored');
  });
});

describe('alertDoor', () => {
  it('filters Alerts to the service', () => {
    expect(alertDoor('orders-shard2-m2')).toEqual({
      label: 'Alert',
      to: '/alerting/status?f.serviceName=orders-shard2-m2',
      context: 'for orders-shard2-m2',
    });
  });
});

describe('sectionAnchor', () => {
  it('keeps the hub params on in-page doors', () => {
    expect(
      sectionAnchor(
        { environment: 'prod', scenario: 'wrong', ops: 'on', role: 'live' },
        'clusters'
      )
    ).toBe('?environment=prod&hubState=wrong&hubOps=on#clusters');
  });

  it('is the bare anchor with default params', () => {
    expect(sectionAnchor(DEFAULT_HUB_PARAMS, 'needs-attention')).toBe(
      '#needs-attention'
    );
  });
});
