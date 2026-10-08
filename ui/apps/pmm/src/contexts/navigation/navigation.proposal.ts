import AppsRounded from '@mui/icons-material/AppsRounded';
import StarBorderRounded from '@mui/icons-material/StarBorderRounded';
import { ColorMode } from '@pmm/shared';
import { CombinedSettings } from 'contexts/settings';
import {
  EXTENSIONS_ATW_PATH,
  EXTENSIONS_MYSQL_BACKUPS_PATH,
  PMM_NEW_NAV_GRAFANA_PATH,
  PROTOTYPE_PLACEHOLDER_PATH,
} from 'lib/constants';
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

// The proposed PMM sidebar (PMM-15353, "Proposal v1" on the IA whiteboard): one
// flat list under section labels instead of per-technology trees. Read top to
// bottom, this file is the sidebar. Every entry is listed for every signed-in
// user so the structure can be reviewed whole; role and feature gating is a
// later step.

const section = (id: string, text: string): NavItem => ({
  id: `section-${id}`,
  type: 'menu-section',
  text,
});

export const NAV_SECTIONS = {
  myNavigation: section('my-navigation', 'My navigation'),
  technologies: section('technologies', 'Technologies'),
  analysis: section('analysis', 'Analysis and alerting'),
  browse: section('browse', 'Browse'),
  administration: section('administration', 'Administration'),
};

// Sample pinned entries: pinning itself is a later iteration, so these stand in
// for what a user would star and let the zone be reviewed with content in it.
export const NAV_MY_NAVIGATION: NavItem[] = [
  {
    id: 'pinned-mongodb-backups',
    icon: StarBorderRounded,
    text: 'MongoDB backups',
    url: `${PMM_NEW_NAV_GRAFANA_PATH}/backup/inventory`,
  },
  {
    id: 'pinned-mysql-innodb-details',
    icon: StarBorderRounded,
    text: 'MySQL InnoDB details',
    url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mysql-innodb/mysql-innodb-details`,
  },
  {
    id: 'pinned-postgresql-query-analytics',
    icon: StarBorderRounded,
    text: 'PostgreSQL query analytics',
    url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/pmm-qan/pmm-query-analytics?var-service_type=postgresql`,
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

export const NAV_QUERY_ANALYTICS: NavItem = {
  ...NAV_QAN,
  text: 'Query analytics',
};

const plannedApp = (slug: string, text: string): NavItem => ({
  id: `app-${slug}`,
  text,
  url: `${PROTOTYPE_PLACEHOLDER_PATH}/${slug}`,
});

// The app catalog as the IA projects it. Two apps exist today and open for
// real; the rest open the placeholder page so the list can be reviewed in full.
export const NAV_APPS: NavItem = {
  id: 'apps',
  icon: AppsRounded,
  text: 'Apps',
  children: [
    plannedApp('data-archiving', 'Data archiving'),
    {
      // Today's backup management, which covers MongoDB; it keeps the badge
      // until the MongoDB backups app replaces it.
      id: 'app-mongodb-backups',
      text: 'MongoDB backups',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/backup/inventory`,
      matches: [`${PMM_NEW_NAV_GRAFANA_PATH}/backup/*`],
      badge: { label: 'Legacy', color: 'default' },
      badgeAlwaysVisible: true,
    },
    {
      id: 'extensions-mysql-backups',
      text: 'MySQL backups',
      url: EXTENSIONS_MYSQL_BACKUPS_PATH,
      matches: ['*'],
    },
    plannedApp('postgresql-backups', 'PostgreSQL backups'),
    plannedApp('proxysql-manager', 'ProxySQL manager'),
    plannedApp('replication-checksums', 'Replication checksums'),
    plannedApp('schema-changes', 'Schema changes'),
    {
      id: 'extensions-atw',
      text: 'Support diagnostics',
      url: EXTENSIONS_ATW_PATH,
      matches: ['*'],
    },
    plannedApp('valkey-backups', 'Valkey backups'),
    plannedApp('get-more-apps', 'Get more apps'),
  ],
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

export const buildProposedNavTree = ({
  user,
  isLoggedIn,
  haInfo,
  settings,
  updateStatus,
  versionInfo,
  colorMode,
  toggleColorMode,
}: ProposedNavTreeInput): NavItem[] => {
  const items: NavItem[] = [
    addHomePage(user?.preferences),
    NAV_SECTIONS.myNavigation,
    ...NAV_MY_NAVIGATION,
    NAV_SECTIONS.technologies,
    ...NAV_TECHNOLOGIES,
    NAV_SECTIONS.analysis,
    NAV_ADVISORS,
    NAV_ALERTS_ENTRY,
    NAV_EXPLORE_DATA,
    NAV_QUERY_ANALYTICS,
    NAV_SECTIONS.browse,
    NAV_DASHBOARDS,
    NAV_APPS,
    NAV_INVENTORY,
    NAV_SECTIONS.administration,
    // Listed whether or not the instance runs in HA, so its place can be
    // reviewed anywhere; a live status badge appears when it does.
    haInfo?.enabled ? addHighAvailability(haInfo) : NAV_HIGH_AVAILABILITY,
    addConfiguration(updateStatus, versionInfo),
    NAV_USERS_AND_ACCESS,
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
