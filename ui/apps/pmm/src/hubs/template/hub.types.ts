import type { SvgIconProps } from '@mui/material/SvgIcon';
import type { ComponentType } from 'react';

export type HubHealth = 'healthy' | 'degraded' | 'down' | 'noData';
export type HubSeverity = 'error' | 'warning' | 'info';
export type HubLevel = 'ok' | 'warning' | 'error';
export type HubScenario = 'live' | 'empty' | 'healthy' | 'wrong';
export type HubOpsMode = 'live' | 'on' | 'off';
export type HubRoleMode = 'live' | 'admin' | 'viewer';
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
  /** `?hubState=` */
  scenario: HubScenario;
  /** `?hubOps=` */
  ops: HubOpsMode;
  /** `?hubRole=` */
  role: HubRoleMode;
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
  /** ISO time the node has been down or without data since. */
  since?: string;
  /** Tooltip, such as "No primary since 14:02". */
  healthDetail?: string;
  members?: { up: number; total: number; noData: number };
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
  /** Worst health; drives the StatusIcon. */
  state: HubHealth | 'empty';
  clusters: number;
  /** Null when metrics failed. */
  needAttention: number | null;
  lastGoodBackup:
    | { at: string; cluster: string }
    | 'none'
    | 'notSetUp'
    | 'featureOff'
    | 'unknown';
  checkedAt: string;
  doors: { clusters: HubDoor; attention: HubDoor; backup: HubDoor };
}

export interface HubBackupRow {
  id: string;
  cluster: string;
  environment?: string;
  /** Level is warning when older than the stale threshold. */
  lastGood: { at: string; level: HubLevel } | null;
  /** Newest failure after the last success. */
  failedAt?: string;
  pitr: 'on' | 'off' | 'stopped' | 'notApplicable';
  nextRun?: string;
  doors: { cluster: HubDoor; lastGood: HubDoor; schedule: HubDoor };
}

export interface HubBackups {
  state: 'ok' | 'notSetUp' | 'featureOff' | 'noServices';
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
  };
  status: HubStatus;
  /** Ranked, uncapped; the view applies the cap. */
  attention: HubAttentionItem[];
  /** Worst first. */
  clusters: HubTreeNode[];
  /** MRT expanded state: every non-healthy node and its ancestors. */
  expanded: Record<string, true>;
  backups: HubBackups;
  queries: HubQueries;
  dashboards: HubDashboards;
  apps: HubApp[];
  actions: { addService?: HubDoor; install?: HubDoor };
  sectionDoors: { attention?: HubDoor; clusters: HubDoor; apps?: HubDoor };
  sections: Record<HubSectionId, HubSectionState>;
}

export interface HubSourceState {
  status: 'disabled' | 'loading' | 'ready' | 'error';
  message?: string;
  updatedAt?: number;
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
  /** The URL's hub params, carried by doors back into the hub; without it they keep only scenario and environment. */
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
  fixtures: Record<
    Exclude<HubScenario, 'live'>,
    (now: number, ops: boolean) => TSnapshot
  >;
  derive: (snapshot: TSnapshot, ctx: HubDeriveContext) => HubViewModel;
}
