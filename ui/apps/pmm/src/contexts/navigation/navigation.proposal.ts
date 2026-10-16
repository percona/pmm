import Add from '@mui/icons-material/Add';
import AppsRounded from '@mui/icons-material/AppsRounded';
import DashboardOutlined from '@mui/icons-material/DashboardOutlined';
import TrackChanges from '@mui/icons-material/TrackChanges';
import { HomeIcon, SupportDiagnosticsIcon } from '@percona/peak-ui';
import { ColorMode } from '@pmm/shared';
import { PreviewChip } from 'components/preview-chip';
import { createElement } from 'react';
import { CombinedSettings } from 'contexts/settings';
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
import { NavItem } from 'types/navigation.types';
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

// Sample pinned entries: pinning itself is a later iteration, so these stand in
// for what a user would star and let the zone be reviewed with content in it.
// Each entry carries the icon of the kind of thing it is, an app or a
// dashboard, and a pinned app leaves the Apps list.
export const NAV_STARRED: NavItem[] = [
  {
    id: 'pinned-mongodb-backups',
    icon: ExtensionIcon,
    text: 'MongoDB backups',
    url: `${PMM_NEW_NAV_GRAFANA_PATH}/backup/inventory`,
    matches: [`${PMM_NEW_NAV_GRAFANA_PATH}/backup/*`],
    pinned: true,
  },
  {
    id: 'pinned-mysql-innodb-details',
    icon: DashboardOutlined,
    text: 'MySQL InnoDB details',
    url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mysql-innodb/mysql-innodb-details`,
    pinned: true,
  },
  {
    id: 'pinned-mysql-backups',
    icon: ExtensionIcon,
    text: 'MySQL backups',
    url: EXTENSIONS_MYSQL_BACKUPS_PATH,
    matches: ['*'],
    pinned: true,
  },
];

const dashboardUrls = (items: NavItem[] = []): string[] =>
  items.flatMap((item) => [
    ...(item.url ? [item.url] : []),
    ...dashboardUrls(item.children),
  ]);

// One entry per technology, meant to open a hub page. Until hubs exist it opens
// the technology's overview dashboard and stays highlighted on any of its
// dashboards, which still live in `navigation.constants`.
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
].map(technology);

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

// The app catalog as the IA projects it. Every entry opens the placeholder page
// so the list can be reviewed in full. Diagnostics is promoted to Analysis and
// alerting and, like the pinned MongoDB and MySQL backups, leaves this list.
export const NAV_APPS: NavItem = {
  id: 'apps',
  icon: AppsRounded,
  text: 'Apps',
  children: [
    plannedApp('data-archiving', 'Data archiving'),
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
    // Not an app but the way to find and add one, so it carries the plus that
    // stands for adding throughout the sidebar.
    { ...plannedApp('get-more-apps', 'Get more apps'), icon: Add },
  ],
};

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
  const { user, isLoggedIn, haInfo, settings, colorMode, toggleColorMode } =
    input;
  const items: NavItem[] = [
    { ...addHomePage(user?.preferences), icon: HomeIcon },
    NAV_SECTIONS.starred,
    ...NAV_STARRED,
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
    NAV_APPS,
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
