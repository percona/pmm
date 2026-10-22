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
export type HubScenario = 'live' | 'empty' | 'healthy' | 'incomplete' | 'wrong';
export type HubOpsMode = 'live' | 'on' | 'off';
export type HubRoleMode = 'live' | 'admin' | 'viewer';
/** Prototype only: `lost` stops sample data refreshing, as a dropped connection would. */
export type HubConnectionMode = 'live' | 'lost';
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
  | 'clusters'
  | 'backups'
  | 'queries'
  | 'dashboards'
  | 'apps';
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
  /** `?hubConnection=` */
  connection: HubConnectionMode;
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
  /** none: no alert rule covers the technology yet; absent when the rules could not be read. */
  alertRules?: 'none' | 'some';
  lastGoodBackup:
    | { at: string; cluster: string }
    | 'none'
    | 'notSetUp'
    | 'featureOff'
    | 'unknown';
  /** Clusters whose newest backup failed lately, or whose point-in-time recovery stopped. */
  backupsFailed: number;
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
  links: { analytics: HubDoor; realtime: HubDoor };
  slower: HubQueryRow[];
  runningLong: HubQueryRow[];
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

export interface HubApp {
  id: string;
  name: string;
  icon: HubIcon;
  preview?: HubPreview;
  /** Figures in the text are doors. */
  summary?: { text: string; doors?: HubDoor[] };
  pages: HubDoor[];
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
  status: HubStatus;
  /** Ranked, uncapped; the view applies the cap. */
  attention: HubAttentionItem[];
  /** Absent when the rules cover what matters, or could not be read. */
  alertCoverage?: HubAlertCoverage;
  /** Worst first; only those behind on patches while that filter is on. */
  clusters: HubTreeNode[];
  /** `?patches=behind` is on. */
  patchesFilter: boolean;
  /** MRT expanded state: every non-healthy node and its ancestors. */
  expanded: Record<string, true>;
  backups: HubBackups;
  queries: HubQueries;
  dashboards: HubDashboards;
  apps: HubApp[];
  actions: {
    /** Ways to bring the technology in, for admins: monitor one that runs, install a new one. */
    add: HubMenuAction[];
    /** Estate-wide pages for admins, then the docs for everyone. */
    more: HubMenuAction[];
  };
  sectionDoors: { attention?: HubDoor; clusters: HubDoor; apps?: HubDoor };
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

/** How current the page's data is. */
export interface HubFreshness {
  /** Epoch ms of the update the label reads: the latest, or the oldest overdue one. */
  at: number;
  /** A source missed its refreshes, as when the connection drops. */
  stale: boolean;
}

export interface HubLiveContext {
  enabled: boolean;
  environment: string | null;
  operations: boolean;
  role: HubRole;
}

export interface HubLiveResult<TSnapshot> {
  snapshot: TSnapshot;
  sources: Record<HubSourceId, HubSourceState>;
  refetch: () => void;
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
  backups: { emptyTitle: string; emptyDescription: string; noServices: string };
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
  useLiveSnapshot: (ctx: HubLiveContext) => HubLiveResult<TSnapshot>;
  /** Sample data per state, built as live sources answer for that environment filter. */
  fixtures: Record<
    Exclude<HubScenario, 'live'>,
    (now: number, ops: boolean, environment: string | null) => TSnapshot
  >;
  derive: (snapshot: TSnapshot, ctx: HubDeriveContext) => HubViewModel;
}
