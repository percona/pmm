import type { SvgIconProps } from '@mui/material/SvgIcon';
import type { ComponentType } from 'react';

export type HubHealth =
  | 'healthy'
  | 'degraded'
  | 'down'
  | 'upgrading'
  | 'noData';
export type HubSeverity = 'error' | 'warning' | 'info';
export type HubLevel = 'ok' | 'warning' | 'error';
export type HubScenario =
  | 'live'
  | 'empty'
  | 'healthy'
  | 'incomplete'
  | 'unhealthy';
export type HubOpsMode = 'live' | 'on' | 'off';
export type HubRoleMode = 'live' | 'admin' | 'viewer';
/** How often the hub refreshes on its own: Grafana's interval set. */
export type HubRefresh =
  | 'off'
  | '5s'
  | '10s'
  | '30s'
  | '1m'
  | '5m'
  | '15m'
  | '30m'
  | '1h'
  | '2h'
  | '1d';
/** Prototype only: `stale` shows data 4 minutes old and stops the real refresh. */
export type HubFreshnessMode = 'live' | 'stale';
export type HubRole = 'admin' | 'viewer';
export type HubSourceId =
  | 'inventory'
  | 'metrics'
  | 'alerts'
  | 'backups'
  | 'qan'
  | 'realtime'
  | 'operations'
  | 'dashboards';
export type HubSectionId =
  | 'attention'
  | 'recommendations'
  | 'clusters'
  | 'backups'
  | 'queries'
  | 'dashboards';
export type HubPreview = 'developer' | 'tech';

// Peak's custom icons are plain components without MUI's muiName, so SvgIconComponent would reject them.
export type HubIcon = ComponentType<SvgIconProps>;

/** Environment value for services without an environment label. */
export const HUB_NO_ENVIRONMENT = '__none__';

export interface HubParams {
  /** `?environment=`; null means every environment. */
  environment: string | null;
  /** `?patches=behind` shows only the clusters a newer patch release exists for. */
  patches: 'behind' | null;
  /** `?hubState=` */
  scenario: HubScenario;
  /** `?hubOps=` */
  ops: HubOpsMode;
  /** `?hubRole=` */
  role: HubRoleMode;
  /** `?refresh=` */
  refresh: HubRefresh;
  /** `?hubFreshness=` */
  freshness: HubFreshnessMode;
}

export interface HubDoor {
  label: string;
  /** Router path below the /pmm-ui basename; Grafana paths start with /graph. */
  to: string;
  /** Visually hidden suffix, such as "for orders-shard1". */
  context?: string;
  preview?: HubPreview;
}

export type HubMeasure =
  | {
      kind: 'value';
      seconds: number;
      level: HubLevel;
      rule?: string;
      door?: HubDoor;
    }
  | { kind: 'notApplicable'; reason: string }
  | { kind: 'noData'; lastSeen?: string };

/** What a version check says about a cluster. */
export type HubUpdate =
  | { state: 'available'; latest: string; checkedAt: string }
  | { state: 'upToDate'; checkedAt: string }
  /** Failed, or too old to trust: never read as up to date. */
  | { state: 'unknown'; reason: 'failed' | 'stale'; checkedAt?: string };

export interface HubVersion {
  /** A member's own version, else the lowest one its reachable members run: what it is still exposed on. */
  effective: string;
  /** Members of one set run different versions, lowest first. */
  drift?: Array<{ version: string; members: string[] }>;
  /** Clusters only; absent while nothing checks for updates. */
  update?: HubUpdate;
}

/** Why a node is not healthy: what holds, and the rule it breaks. */
export interface HubCondition {
  /** "billing-m2 47s behind" */
  text: string;
  /** "Degraded when a secondary is 10s or more behind, sustained for 2m" */
  rule: string;
}

export interface HubTreeColumn {
  /** Key in HubTreeNode.measures. */
  id: string;
  header: string;
  /** Header tooltip. */
  description: string;
}

export interface HubTreeNode {
  /** cluster:<env>/<name> | group:<env>/<cluster>/routers | set:<env>/<cluster>/<rs> | member:<serviceId> */
  id: string;
  kind: 'cluster' | 'group' | 'set' | 'member';
  name: string;
  /** "Sharded cluster", "Replica set", "Standalone", "Routers", "Config servers", "Shard", "Primary", "Secondary", "Arbiter", "Router", "Member" */
  type: string;
  environment?: string;
  health: HubHealth;
  /** False on members that are up. */
  showHealth: boolean;
  /** ISO time the node has been down, without data or upgrading since. */
  since?: string;
  /** What makes it other than healthy, the deciding one first; none that only repeat its health word. */
  conditions?: HubCondition[];
  members?: { up: number; total: number; noData: number };
  version?: HubVersion;
  measures: Record<string, HubMeasure>;
  doors: { name: HubDoor; members?: HubDoor; menu: HubDoor[] };
  children?: HubTreeNode[];
}

/** A problem a technology's rules find in live data, before the hub turns it into a Needs attention row. */
export interface HubAttentionItem {
  /** `${kind}:${subjectId}`, stable across refetches. */
  id: string;
  /** Technology-defined, ranked by config.attentionOrder. */
  kind: string;
  severity: HubSeverity;
  title: string;
  where: string;
  detail?: string;
  /** ISO */
  since?: string;
  door: HubDoor;
  /** The merged firing alert ("Alert"). */
  secondaryDoor?: HubDoor;
  source?: 'alert' | 'operations';
}

/** How bad a Needs attention row is, worst first. */
export type HubAttentionSeverity = 'critical' | 'error' | 'warning';

/** Where a Needs attention row's action goes. */
export interface HubDestination {
  /** "ReplSet summary" */
  label: string;
  /** A router path below /pmm-ui ("/graph/d/…"), a section of this page ("#clusters"), or an absolute URL. */
  route: string;
  /** Query parameters, such as Grafana's `var-cluster`. */
  params?: Record<string, string>;
  /** page: another PMM page (→); inpage: further down this hub (↓); external: outside PMM (↗). */
  kind: 'page' | 'inpage' | 'external';
}

/** A recommendation's kind: something not set up yet, or something worth knowing. Not set up comes first. */
export type HubRecommendationKind = 'notset' | 'info';

/** One Recommendations row: fine today, worth improving. The Needs attention format, for a steady state, so with no duration and no alert. */
export interface HubRecommendation {
  kind: HubRecommendationKind;
  title: string;
  impact: string;
  /** The clusters: "orders-us, cart-us and 7 more", or "All clusters". */
  location: string;
  /** Only when every affected cluster is in this one. */
  environment?: string;
  clusterCount: number;
  /** Where the recommendation comes from; not shown, kept for the requirements. */
  source: string;
  destination: HubDestination;
}

/** How the hub draws a destination: the sidebar's icon for it, and where it sits in the sidebar. */
export interface HubDestinationLook {
  icon?: HubIcon;
  /** ["Apps", "Operations for MongoDB", "Fleet"] */
  path: string[];
}

/** One Needs attention row: a problem that is true right now. The sample data is written in this format, so it doubles as the requirements. */
export interface HubNeedsAttentionItem {
  severity: HubAttentionSeverity;
  /** What's wrong and where: "billing-eu has no primary and can’t accept writes". */
  title: string;
  /** Why it matters, in plain words. */
  impact: string;
  /** A cluster, a path ("orders-eu › orders-eu-shard2") or a list ("a, b and 3 more"). */
  location: string;
  /** "prod", or "prod, staging" for a row that spans environments. */
  environment: string;
  /** A duration for a condition ("18m", "up to 40m"), or an event ("8h ago"). */
  since: string;
  /** The underlying issues the row stands for; the section's count adds these up. */
  issueCount: number;
  alertFiring: boolean;
  /** The service the firing alert is about, which the Alerts link filters on. */
  alertService?: string;
  /** Where the problem comes from; not shown, kept for the requirements. */
  source: string;
  destination: HubDestination;
}

export interface HubStatus {
  /** Worst health of the shown clusters. */
  state: HubHealth | 'empty';
  /** Null when the inventory failed. */
  clusters: number | null;
  /** Shown clusters by health; absent when metrics or the inventory failed. */
  health?: Record<HubHealth, number>;
  /** Clusters with an update available, and those the check could not answer for; absent while nothing checks. */
  patches?: { behind: number; unknown: number };
  /** Firing alerts about the technology or its hosts; null when they could not be read, absent while alerting is off. */
  alerts?: number | null;
  /** How many of the firing alerts are critical. */
  alertsCritical?: number;
  /** none: no alert rule covers the technology yet; absent when the rules could not be read. */
  alertRules?: 'none' | 'some';
  lastGoodBackup:
    | { at: string; cluster: string }
    | 'none'
    | 'notSetUp'
    | 'featureOff'
    | 'unknown';
  /** Clusters whose newest backup failed lately. */
  backupsFailed: number;
  /** The oldest of those clusters' last good backups, ISO; absent when none of them has one. */
  backupsFailedOldestGood?: string;
  /** Clusters a backup tool can protect (not standalones), and those with an enabled schedule; absent until backups are read. */
  backupCoverage?: { protected: number; protectable: number };
  checkedAt: string;
  /** Where to look into firing alerts and failed backups, and to set up what is missing (admins only). */
  doors: {
    alerts: HubDoor;
    backups: HubDoor;
    setUpAlerts?: HubDoor;
    setUpBackups?: HubDoor;
  };
}

/** Which key conditions no alert rule covers, and where to create the rules. */
export interface HubAlertCoverage {
  /** "No alert rules cover MongoDB yet." */
  text: string;
  /** Create from templates; admins only. */
  door?: HubDoor;
  /** After the door: the templates to start from. */
  detail?: string;
}

export interface HubBackupRow {
  id: string;
  cluster: string;
  environment?: string;
  /** Level is warning when older than the stale threshold. */
  lastGood: { at: string; level: HubLevel } | null;
  /** Newest failure after the last success. */
  failedAt?: string;
  /** notStarted: an enabled schedule that has no artifact yet, so nothing says it works. */
  pitr: 'on' | 'off' | 'stopped' | 'notStarted' | 'notApplicable';
  nextRun?: string;
  doors: { cluster: HubDoor; lastGood: HubDoor; schedule: HubDoor };
}

export interface HubBackups {
  /** unknown: the source failed or has not answered, so nothing can be said. */
  state: 'ok' | 'notSetUp' | 'featureOff' | 'noServices' | 'unknown';
  /** Problems first, then no backup, then by name. */
  rows: HubBackupRow[];
  /** "PBM is not configured on billing, orders" */
  hint?: string;
  doors: {
    app: HubDoor;
    dashboard: HubDoor;
    setUp?: HubDoor;
    settings?: HubDoor;
  };
}

export interface HubQueryRow {
  id: string;
  /** Fingerprint or command, truncated in the UI. */
  text: string;
  /** "6× slower than usual" | "running for 14m" */
  headline: string;
  /** "0.84s avg · usual 0.14s · 1.2k calls in 1h" */
  detail: string;
  severity: HubSeverity;
  door: HubDoor;
}

export interface HubQueries {
  slower: HubQueryRow[];
}

export interface HubDashboardLink extends HubDoor {
  uid: string;
  visitedAt?: string;
}

export interface HubDashboards {
  recommended: HubDashboardLink[];
  forDevelopers: HubDashboardLink[];
  /** Newest first, at most 5. */
  recent: HubDashboardLink[];
  all: HubDoor & { count?: number };
}

/** An entry of a header menu: Add MongoDB, or More actions. */
export interface HubMenuAction extends HubDoor {
  icon: HubIcon;
  /** A second line under the label, such as "Connect PMM to a database you already run". */
  description?: string;
  /** Opens outside PMM, in a new tab. */
  external?: boolean;
}

export interface HubCapabilities {
  /** Role admin. */
  addService: boolean;
  /** Role admin and operations. */
  install: boolean;
  /** isPMMAdmin and omEnabled (or hubOps=on in fixtures). */
  operations: boolean;
  /** Role admin and backup management on. */
  setUpBackup: boolean;
  /** Signed in, not anonymous, extensionsEnabled. */
  extensions: boolean;
}

export interface HubSectionState {
  status: 'ready' | 'loading' | 'error';
  message?: string;
  source?: HubSourceId;
  /** The failed source is one of several, so the section still shows what the others gave. */
  partial?: boolean;
}

export interface HubViewModel {
  technology: { id: string; name: string };
  scenario: HubScenario;
  role: HubRole;
  capabilities: HubCapabilities;
  environment: {
    selected: string | null;
    /** Each environment, then "No environment"; the select adds "All environments" itself. */
    options: Array<{ value: string; label: string }>;
    /** A filter is set and no service is in it, which is not an empty estate. */
    matchesNothing: boolean;
  };
  /** The places the hub leads to most, as tiles under the header, in order; the same in every state. */
  shortcuts: HubDestination[];
  status: HubStatus;
  /** Worst first, then by impact, then longest first; uncapped, as the view shows a few and folds the rest. */
  attention: HubNeedsAttentionItem[];
  /** Absent when the rules cover what matters, or could not be read; Recommendations says it. */
  alertCoverage?: HubAlertCoverage;
  /** Not set up first, then info, each in its order: backups and recovery, security, versions and patches, monitoring coverage. */
  recommendations: HubRecommendation[];
  /** Worst first; only those behind on patches while that filter is on. */
  clusters: HubTreeNode[];
  /** `?patches=behind` is on. */
  patchesFilter: boolean;
  /** MRT expanded state; every cluster starts as one row. */
  expanded: Record<string, true>;
  backups: HubBackups;
  queries: HubQueries;
  dashboards: HubDashboards;
  actions: {
    /** Ways to bring the technology in, for admins: monitor one that runs, install a new one. */
    add: HubMenuAction[];
    /** Estate-wide pages for admins, then the docs for everyone. */
    more: HubMenuAction[];
  };
  /** The index of every app, beside Dashboards' own door. */
  sectionDoors: { clusters: HubDoor; apps?: HubDoor };
  sections: Record<HubSectionId, HubSectionState>;
}

export interface HubSourceState {
  status: 'disabled' | 'loading' | 'ready' | 'error';
  message?: string;
  /** The last successful fetch, kept when a refresh fails. */
  updatedAt?: number;
  /** How often it refreshes, in ms; absent for a source that refreshes only on demand. */
  refreshEvery?: number;
}

/** Refreshing on its own, refreshing only on demand, or behind on its refreshes. */
export type HubFreshnessState = 'live' | 'off' | 'stale';

/** How current the page's data is. */
export interface HubFreshness {
  state: HubFreshnessState;
  /** Epoch ms of the last good update: the latest, or the oldest overdue one; null before any. */
  at: number | null;
  /** The auto-refresh interval the reader picked. */
  interval: HubRefresh;
  /** A refresh is in flight. */
  refreshing: boolean;
}

export interface HubLiveContext {
  enabled: boolean;
  /** How often scheduled sources refresh, in ms; false stops them. */
  refreshEvery: number | false;
  environment: string | null;
  operations: boolean;
  role: HubRole;
}

export interface HubLiveResult<TSnapshot> {
  snapshot: TSnapshot;
  sources: Record<HubSourceId, HubSourceState>;
  refetch: () => void;
  /** A scheduled or requested refresh is in flight. */
  fetching: boolean;
}

export interface HubDeriveContext {
  now: number;
  /** The URL's hub params; without them, only the scenario and environment are known. */
  params?: HubParams;
  scenario: HubScenario;
  environment: string | null;
  role: HubRole;
  operations: boolean;
  capabilities: HubCapabilities;
  sources: Record<HubSourceId, HubSourceState>;
}

export interface HubMessages {
  /** "MongoDB" */
  name: string;
  /** ["cluster", "clusters"] */
  unit: { cluster: [string, string] };
  empty: { title: string; description: string; viewer: string };
  /** tableLabel: "MongoDB clusters" */
  clusters: { tableLabel: string };
  /** all: "All MongoDB dashboards" */
  dashboards: { all: string };
}

export interface TechnologyHubConfig<TSnapshot> {
  /** Grows per hub. */
  id: 'mongodb';
  /** The sidebar entry's id, NAV_MONGO.id ('mongo'). */
  navId: string;
  path: string;
  name: string;
  icon: HubIcon;
  messages: HubMessages;
  treeColumns: HubTreeColumn[];
  /** Kinds, highest priority first. */
  attentionOrder: string[];
  /** The sidebar's icon for a destination of a row or a shortcut tile, and its place there. */
  describeDestination: (destination: HubDestination) => HubDestinationLook;
  useLiveSnapshot: (ctx: HubLiveContext) => HubLiveResult<TSnapshot>;
  /** Sample data per state, built as live sources answer for that environment filter. */
  fixtures: Record<
    Exclude<HubScenario, 'live'>,
    (now: number, ops: boolean, environment: string | null) => TSnapshot
  >;
  derive: (snapshot: TSnapshot, ctx: HubDeriveContext) => HubViewModel;
}
