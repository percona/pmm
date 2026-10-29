import AppRegistrationRounded from '@mui/icons-material/AppRegistrationRounded';
import AppsRounded from '@mui/icons-material/AppsRounded';
import DashboardOutlined from '@mui/icons-material/DashboardOutlined';
import TrackChanges from '@mui/icons-material/TrackChanges';
import { HomeIcon, SupportDiagnosticsIcon } from '@percona/peak-ui';
import { ColorMode } from '@pmm/shared';
import { PreviewChip } from 'components/preview-chip';
import { createElement } from 'react';
import { CombinedSettings } from 'contexts/settings';
import { withHubPage } from 'hubs/hubs.paths';
import {
  EXTENSIONS_ATW_PATH,
  EXTENSIONS_MYSQL_BACKUPS_PATH,
  OM_PATH,
  PMM_NEW_NAV_GRAFANA_PATH,
  PMM_NEW_NAV_PATH,
  PROTOTYPE_PLACEHOLDER_PATH,
} from 'lib/constants';
import { ExtensionIcon } from 'lib/peak-ui-pending-icons';
import { HAInfo } from 'types/ha.types';
import type { StarredDashboard } from 'types/dashboards.types';
import { NavItem } from 'types/navigation.types';
import { findActiveNavItem } from 'utils/navigation.utils';
import { GetUpdatesResponse, UpdateStatus } from 'types/updates.types';
import { User } from 'types/user.types';
import {
  NAV_ADVISORS,
  NAV_ALERTS,
  NAV_ALERTS_CONTACT_POINTS,
  NAV_ALERTS_GROUPS,
  NAV_ALERTS_NOTIFICATION_POLICIES,
  NAV_ALERTS_RULES,
  NAV_ALERTS_SETTINGS,
  NAV_ALERTS_SILENCES,
  NAV_ALERTS_STATUS,
  NAV_ALERTS_TEMPLATES,
  NAV_DASHBOARDS,
  NAV_EXPLORE,
  NAV_HELP,
  NAV_HIGH_AVAILABILITY,
  NAV_INVENTORY,
  NAV_MONGO,
  NAV_MYSQL,
  NAV_OS,
  NAV_POSTGRESQL,
  NAV_QAN,
  NAV_SIGN_IN,
  NAV_USERS_AND_ACCESS,
  NAV_VALKEY,
} from './navigation.constants';
import {
  addAccount,
  addConfiguration,
  addHighAvailability,
  addHomePage,
} from './navigation.utils';

// The proposed PMM sidebar (PMM-15353, "Proposal v2" on the IA whiteboard): one
// flat list under section labels instead of per-technology trees. Read top to
// bottom, this file is the sidebar. Every entry is listed for every signed-in
// user so the structure can be reviewed whole; role and feature gating is a
// later step.

const section = (id: string, text: string) => ({
  id: `section-${id}`,
  type: 'menu-section' as const,
  text,
});

export const NAV_SECTIONS = {
  starred: section('starred', 'Starred'),
  technologies: section('technologies', 'Technologies'),
  analysis: section('analysis', 'Analysis and alerting'),
  browse: section('browse', 'Browse'),
  administration: section('administration', 'Administration'),
};

// Apps are starred from their header and dashboards in Grafana. Each entry
// carries the icon of the kind of thing it is, an app or a dashboard, and a
// starred app leaves the Apps list. A first visit starts with the backups
// apps starred.
export const DEFAULT_STARRED_APPS = [
  'app-mongodb-backups',
  'app-mysql-backups',
];

// A starred app: the puzzle piece, unless its developer gave it an icon.
export const starredApp = (app: NavItem): NavItem => ({
  ...app,
  icon: app.icon ?? ExtensionIcon,
  pinned: true,
});

// A dashboard the user starred in Grafana. The entry follows the dashboard's
// uid, so it stays highlighted whatever the address's slug or query.
export const starredDashboard = ({
  uid,
  title,
  url,
}: StarredDashboard): NavItem => ({
  id: `starred-dashboard-${uid}`,
  icon: DashboardOutlined,
  text: title,
  url,
  matches: [`${PMM_NEW_NAV_GRAFANA_PATH}/d/${uid}/*`],
  pinned: true,
});

const dashboardUrls = (items: NavItem[] = []): string[] =>
  items.flatMap((item) => [
    ...(item.url ? [item.url] : []),
    ...dashboardUrls(item.children),
  ]);

// One entry per technology: its hub when one exists, else its overview
// dashboard, highlighted on any of its dashboards in `navigation.constants`.
const technology = ({ id, text, icon, url, children }: NavItem): NavItem => ({
  id,
  text,
  icon,
  url,
  matches: dashboardUrls(children),
});

export const NAV_TECHNOLOGIES: NavItem[] = [
  NAV_MONGO,
  NAV_MYSQL,
  NAV_POSTGRESQL,
  NAV_VALKEY,
  NAV_OS,
]
  .map(technology)
  .map(withHubPage);

export const NAV_ALERTS_ENTRY: NavItem = {
  ...NAV_ALERTS,
  children: [
    NAV_ALERTS_STATUS,
    NAV_ALERTS_RULES,
    NAV_ALERTS_TEMPLATES,
    NAV_ALERTS_CONTACT_POINTS,
    NAV_ALERTS_NOTIFICATION_POLICIES,
    NAV_ALERTS_SILENCES,
    NAV_ALERTS_GROUPS,
    NAV_ALERTS_SETTINGS,
  ],
};

export const NAV_EXPLORE_DATA: NavItem = {
  ...NAV_EXPLORE,
  text: 'Explore data',
  matches: [
    ...(NAV_EXPLORE.matches || []),
    `${PMM_NEW_NAV_GRAFANA_PATH}/explore/*`,
  ],
};

// Stored query metrics and live query sessions are separate jobs, so each is
// its own entry rather than a tab inside one Query analytics page.
export const NAV_QUERY_ANALYTICS: NavItem = {
  ...NAV_QAN,
  text: 'Query analytics',
  matches: ['*'],
};

export const NAV_REALTIME_QUERIES: NavItem = {
  id: 'realtime-queries',
  icon: TrackChanges,
  text: 'Real-time queries',
  url: `${PMM_NEW_NAV_PATH}/rta`,
  matches: ['*'],
};

const plannedApp = (slug: string, text: string): NavItem => ({
  id: `app-${slug}`,
  text,
  url: `${PROTOTYPE_PLACEHOLDER_PATH}/${slug}`,
});

// The app catalog as the IA projects it: every app under Apps, each of which
// can be starred. The planned ones open the placeholder page so the list can
// be reviewed in full. Diagnostics, Query analytics and Real-time queries have
// places of their own in the sidebar instead.
export const APPS: NavItem[] = [
  plannedApp('data-archiving', 'Data archiving'),
  {
    id: 'app-mongodb-backups',
    text: 'MongoDB backups',
    url: `${PMM_NEW_NAV_GRAFANA_PATH}/backup/inventory`,
    matches: [`${PMM_NEW_NAV_GRAFANA_PATH}/backup/*`],
  },
  {
    id: 'app-mysql-backups',
    text: 'MySQL backups',
    url: EXTENSIONS_MYSQL_BACKUPS_PATH,
    matches: ['*'],
  },
  // A real app, listed here rather than under MongoDB. Its own tabs carry
  // Fleet, Nodes, Automations and Settings, so it needs one entry,
  // highlighted on all of them.
  {
    id: 'app-operations-for-mongodb',
    text: 'Operations for MongoDB',
    url: OM_PATH,
    matches: ['*'],
    badge: createElement(PreviewChip, { level: 'developer' }),
    badgeAlwaysVisible: true,
  },
  plannedApp('postgresql-backups', 'PostgreSQL backups'),
  plannedApp('proxysql-manager', 'ProxySQL manager'),
  plannedApp('replication-checksums', 'Replication checksums'),
  plannedApp('schema-changes', 'Schema changes'),
  // A fictional Technical Preview: still maturing and not production-ready.
  // The chip is a label rather than news, so it puts no dot on the closed
  // group.
  {
    ...plannedApp('valkey-backups', 'Valkey backups'),
    badge: createElement(PreviewChip, { level: 'tech' }),
    badgeAlwaysVisible: true,
  },
];

// Not an app but the index of them all, installed and available. Named like
// All dashboards, which does the same for dashboards.
export const NAV_ALL_APPS: NavItem = {
  ...plannedApp('all-apps', 'All apps'),
  icon: AppRegistrationRounded,
};

// The Apps group: the apps not starred, then the index of them all.
export const appsEntry = (starredApps: string[] = []): NavItem => ({
  id: 'apps',
  icon: AppsRounded,
  text: 'Apps',
  children: [
    ...APPS.filter((app) => !starredApps.includes(app.id)),
    NAV_ALL_APPS,
  ],
});

// The ATW app runs diagnostic checks against monitored databases; sending the
// results to a Percona Support case is one action inside it, not its purpose.
// Named for the job rather than the audience, it sits with the other analysis
// tools. The Help page keeps a card for the "Support asked me for a bundle"
// route, which is where the tree test showed that task being looked for.
export const NAV_DIAGNOSTICS: NavItem = {
  id: 'extensions-atw',
  icon: SupportDiagnosticsIcon,
  text: 'Diagnostics',
  url: EXTENSIONS_ATW_PATH,
  matches: ['*'],
};

export interface ProposedNavTreeInput {
  user?: User;
  isLoggedIn: boolean;
  haInfo?: HAInfo;
  settings?: CombinedSettings;
  updateStatus: UpdateStatus;
  versionInfo?: GetUpdatesResponse;
  colorMode: ColorMode;
  toggleColorMode: () => void;
  starredDashboards?: StarredDashboard[];
  // The ids of the starred apps, in the order they were starred.
  starredApps?: string[];
}

// Configuration is the one place for everything configurable about PMM, so
// Users and access folds into it. Org. management and Users and access are
// groups PMM inherits from Grafana's user management, and keeping them as
// groups would make Configuration the only place with a fold inside a fold.
// Their pages are listed directly instead, under headings, so the whole
// sidebar stays one level deep. The Account stays a top-level entry: it
// carries the signed-in user's name, which is how people see who is logged
// in, and that is lost one level down.
const addConfigurationGroup = ({
  updateStatus,
  versionInfo,
}: ProposedNavTreeInput): NavItem => {
  const configuration = addConfiguration(updateStatus, versionInfo);
  const children = configuration.children || [];
  const orgManagement = children.find((child) => child.id === 'org-management');

  return {
    ...configuration,
    children: [
      ...children.filter((child) => child !== orgManagement),
      section('organization', 'Organization'),
      ...(orgManagement?.children || []),
      section('users-and-access', 'Users and access'),
      ...(NAV_USERS_AND_ACCESS.children || []),
    ],
  };
};

export const buildProposedNavTree = (
  input: ProposedNavTreeInput
): NavItem[] => {
  const {
    user,
    isLoggedIn,
    haInfo,
    settings,
    colorMode,
    toggleColorMode,
    starredDashboards = [],
    starredApps = [],
  } = input;
  const items: NavItem[] = [
    { ...addHomePage(user?.preferences), icon: HomeIcon },
    NAV_SECTIONS.starred,
    ...starredApps.flatMap((id) =>
      APPS.filter((app) => app.id === id).map(starredApp)
    ),
    ...starredDashboards.map(starredDashboard),
    NAV_SECTIONS.technologies,
    ...NAV_TECHNOLOGIES,
    NAV_SECTIONS.analysis,
    NAV_ADVISORS,
    NAV_ALERTS_ENTRY,
    NAV_DIAGNOSTICS,
    NAV_EXPLORE_DATA,
    NAV_QUERY_ANALYTICS,
    NAV_REALTIME_QUERIES,
    NAV_SECTIONS.browse,
    NAV_DASHBOARDS,
    appsEntry(starredApps),
    NAV_INVENTORY,
    NAV_SECTIONS.administration,
    // Listed whether or not the instance runs in HA, so its place can be
    // reviewed anywhere; a live status badge appears when it does.
    haInfo?.enabled ? addHighAvailability(haInfo) : NAV_HIGH_AVAILABILITY,
    addConfigurationGroup(input),
  ];

  if (user && isLoggedIn) {
    items.push(addAccount(user, colorMode, toggleColorMode, settings));
  }

  items.push(NAV_HELP);

  if (!isLoggedIn) {
    items.push(NAV_SIGN_IN);
  }

  return items;
};

// Apps with places of their own in the sidebar, and the index of all apps:
// their headers share a link but cannot star them.
const FIXED_APPS: NavItem[] = [
  NAV_DIAGNOSTICS,
  NAV_QUERY_ANALYTICS,
  NAV_REALTIME_QUERIES,
  NAV_ALL_APPS,
];

export interface AppPage {
  app: NavItem;
  starrable: boolean;
}

// The app a page belongs to, for its header; undefined outside any app.
export const findAppPage = (pathname: string): AppPage | undefined => {
  const app = findActiveNavItem(APPS, pathname);

  if (app) {
    return { app, starrable: true };
  }
  const fixed = findActiveNavItem(FIXED_APPS, pathname);

  return fixed && { app: fixed, starrable: false };
};
