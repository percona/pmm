import type { HubHealth, HubLevel } from 'hubs/template/hub.types';
import type { MongoPlace } from './mongodb.doors';
import {
  hasNoPrimary,
  type HealthVerdict,
  lagLevel,
  type MemberHealthInput,
  type MemberVerdict,
  memberVerdict,
  oplogLevel,
  routersHealth,
  setHealth,
  shardedHealth,
  standaloneHealth,
  upgradingHealth,
} from './mongodb.health';
import { Messages } from './mongodb.messages';
import type {
  MongoMetrics,
  MongoServiceInput,
  MongoUpgradeInput,
} from './mongodb.types';

export interface MemberModel {
  service: MongoServiceInput;
  verdict: MemberVerdict;
  /** member_state while up: PRIMARY, SECONDARY, ARBITER and so on. */
  state?: string;
  /** How long the member has been in its state, as far as the last hour tells. */
  stateSeconds?: number;
  /** The server version it reported last, such as 7.0.24-13. */
  version?: string;
  router: boolean;
  /** Epoch seconds the member was last seen up. */
  lastUpAt?: number;
  lag?: { now: number; sustained: number };
  oplogWindow?: number;
  oplogFill?: number;
}

export interface SetModel {
  /** Exporter's replica set name (rs_nm), else PMM's replication_set label. */
  name: string;
  config: boolean;
  members: MemberModel[];
}

export interface ClusterModel {
  /** `${environment ?? ''}/${name}` */
  key: string;
  environment?: string;
  /** PMM's cluster label, else the replica set or service name the services group by. */
  name: string;
  /** PMM's cluster label; absent when the services carry none, so doors leave var-cluster out. */
  label?: string;
  kind: 'sharded' | 'replicaSet' | 'standalone';
  routers: MemberModel[];
  config?: SetModel;
  /** Shards of a sharded cluster, or the one set of a replica set. */
  sets: SetModel[];
  standalone?: MemberModel;
  /** An upgrade Operations runs on the cluster now. */
  upgrade?: MongoUpgradeInput;
}

export const clusterKey = (environment: string | undefined, name: string) =>
  `${environment ?? ''}/${name}`;

const toMember = (
  service: MongoServiceInput,
  metrics: MongoMetrics | null,
  now: number
): MemberModel => {
  const verdict = memberVerdict(service, metrics);
  const id = service.serviceId;
  const self = metrics?.self[id];
  const stateSince = metrics?.stateSince[id];

  return {
    service,
    verdict,
    state: verdict === 'up' ? self?.state : undefined,
    ...(verdict === 'up' && self?.state && stateSince !== undefined
      ? { stateSeconds: Math.max(0, Math.round(now / 1000 - stateSince)) }
      : {}),
    ...(metrics?.versions[id] ? { version: metrics.versions[id] } : {}),
    router: metrics?.up[id]?.clusterRole === 'mongos',
    lastUpAt: metrics?.lastUpAt[id],
    lag: metrics?.lag[id],
    oplogWindow: metrics?.oplogWindow[id],
    oplogFill: metrics?.oplogFill[id],
  };
};

const setNameOf = (member: MemberModel, metrics: MongoMetrics | null) =>
  metrics?.self[member.service.serviceId]?.rsName ||
  metrics?.roles[member.service.serviceId]?.rsName ||
  member.service.replicationSet ||
  undefined;

// A member that is down reports no role, so the one it had in the last day places its set.
const isConfigServer = (member: MemberModel, metrics: MongoMetrics | null) =>
  (metrics?.self[member.service.serviceId]?.clRole ??
    metrics?.roles[member.service.serviceId]?.clRole) === 'configsvr';

/** Topology order: primary, secondaries, arbiters, then members without a reported state. */
const STATE_ORDER: Record<string, number> = {
  PRIMARY: 0,
  SECONDARY: 1,
  ARBITER: 2,
};

const byTopology = (a: MemberModel, b: MemberModel) =>
  (STATE_ORDER[a.state ?? ''] ?? 3) - (STATE_ORDER[b.state ?? ''] ?? 3) ||
  a.service.serviceName.localeCompare(b.service.serviceName);

const byName = (a: { name: string }, b: { name: string }) =>
  a.name.localeCompare(b.name);

/** Groups services by environment and cluster (else replication set); routers, config servers or two sets make it sharded. */
export const groupClusters = (
  services: MongoServiceInput[],
  metrics: MongoMetrics | null,
  now = Date.now()
): ClusterModel[] => {
  const groups = new Map<
    string,
    {
      environment?: string;
      name: string;
      label?: string;
      members: MemberModel[];
    }
  >();
  const standalones: ClusterModel[] = [];

  for (const service of services) {
    const name = service.cluster || service.replicationSet;
    const member = toMember(service, metrics, now);

    if (!name) {
      standalones.push({
        key: clusterKey(service.environment, service.serviceName),
        environment: service.environment,
        name: service.serviceName,
        kind: 'standalone',
        routers: [],
        sets: [],
        standalone: member,
      });
      continue;
    }
    const key = clusterKey(service.environment, name);
    const group = groups.get(key) ?? {
      environment: service.environment,
      name,
      members: [],
    };

    group.label ??= service.cluster || undefined;
    group.members.push(member);
    groups.set(key, group);
  }

  const clusters = [...groups.entries()].flatMap(
    ([key, { environment, name, label, members }]): ClusterModel[] => {
      const routers = members.filter((member) => member.router);
      const sets = new Map<string, SetModel>();
      const unplaced: MemberModel[] = [];

      for (const member of members) {
        if (member.router) {
          continue;
        }
        const setName = setNameOf(member, metrics);

        if (!setName) {
          unplaced.push(member);
          continue;
        }
        const set = sets.get(setName) ?? {
          name: setName,
          config: false,
          members: [],
        };

        set.config ||= isConfigServer(member, metrics);
        set.members.push(member);
        sets.set(setName, set);
      }

      const config = [...sets.values()].find((set) => set.config);
      const dataSets = [...sets.values()].filter((set) => !set.config);
      const sharded = routers.length > 0 || !!config || dataSets.length > 1;

      // A member without a set name in a sharded cluster is a router whose exporter is silent.
      if (sharded) {
        routers.push(...unplaced);
      } else if (dataSets.length === 1) {
        dataSets[0].members.push(...unplaced);
      } else {
        return unplaced.map((member) => ({
          key: clusterKey(environment, member.service.serviceName),
          environment,
          name: unplaced.length === 1 ? name : member.service.serviceName,
          ...(label ? { label } : {}),
          kind: 'standalone',
          routers: [],
          sets: [],
          standalone: member,
        }));
      }

      [...sets.values()].forEach((set) => set.members.sort(byTopology));
      routers.sort(byTopology);

      return [
        {
          key,
          environment,
          name,
          ...(label ? { label } : {}),
          kind: sharded ? 'sharded' : 'replicaSet',
          routers,
          config,
          sets: dataSets.sort(byName),
        },
      ];
    }
  );

  return [...clusters, ...standalones];
};

/** Every member of a cluster, routers and config servers included. */
export const clusterMembers = (cluster: ClusterModel): MemberModel[] => [
  ...cluster.routers,
  ...(cluster.config?.members ?? []),
  ...cluster.sets.flatMap((set) => set.members),
  ...(cluster.standalone ? [cluster.standalone] : []),
];

export const primaryOf = (set: SetModel) =>
  set.members.find(
    (member) => member.verdict === 'up' && member.state === 'PRIMARY'
  );

const isSecondary = (member: MemberModel) =>
  member.verdict === 'up' && member.state === 'SECONDARY';

const isPrimary = (member: MemberModel) =>
  member.verdict === 'up' && member.state === 'PRIMARY';

export const memberLagLevel = (member: MemberModel): HubLevel =>
  member.lag && isSecondary(member) ? lagLevel(member.lag.sustained) : 'ok';

export const primaryOplogLevel = (set: SetModel): HubLevel | undefined => {
  const primary = primaryOf(set);

  return primary?.oplogWindow !== undefined
    ? oplogLevel(primary.oplogWindow, primary.oplogFill)
    : undefined;
};

/** What the health rules read of a member; lag and oplog only count inside a set. */
const healthInput = (
  member: MemberModel,
  inSet = false
): MemberHealthInput => ({
  name: member.service.serviceName,
  verdict: member.verdict,
  ...(member.state ? { state: member.state } : {}),
  ...(member.stateSeconds !== undefined
    ? { stateSeconds: member.stateSeconds }
    : {}),
  ...(member.version ? { version: member.version } : {}),
  ...(inSet && member.lag && isSecondary(member) ? { lag: member.lag } : {}),
  ...(inSet && member.oplogWindow !== undefined && isPrimary(member)
    ? {
        oplog: {
          seconds: member.oplogWindow,
          ...(member.oplogFill !== undefined ? { fill: member.oplogFill } : {}),
        },
      }
    : {}),
});

export const setVerdictOf = (set: SetModel): HealthVerdict =>
  setHealth(
    set.members.map((member) => healthInput(member, true)),
    set.name
  );

export const setHealthOf = (set: SetModel): HubHealth =>
  setVerdictOf(set).health;

const routersVerdictOf = (cluster: ClusterModel): HealthVerdict =>
  routersHealth(
    cluster.routers.map((member) => healthInput(member)),
    Messages.routersName
  );

/** A cluster's health and the conditions behind it; an upgrade in progress decides before anything else. */
export const clusterVerdictOf = (cluster: ClusterModel): HealthVerdict => {
  const own =
    cluster.kind === 'standalone' && cluster.standalone
      ? standaloneHealth(healthInput(cluster.standalone))
      : cluster.kind === 'replicaSet'
        ? setVerdictOf(cluster.sets[0])
        : shardedHealth({
            shards: cluster.sets.map(setVerdictOf),
            config: cluster.config ? setVerdictOf(cluster.config) : undefined,
            routers: cluster.routers.length
              ? routersVerdictOf(cluster)
              : undefined,
          });

  return cluster.upgrade ? upgradingHealth(own, cluster.upgrade) : own;
};

export const clusterHealthOf = (cluster: ClusterModel): HubHealth =>
  clusterVerdictOf(cluster).health;

export const setHasNoPrimary = (set: SetModel) => hasNoPrimary(set.members);

export const placeOf = (cluster: ClusterModel): MongoPlace => ({
  environment: cluster.environment,
  cluster: cluster.label,
});
