import AppsRounded from '@mui/icons-material/AppsRounded';
import TrackChanges from '@mui/icons-material/TrackChanges';
import { SupportDiagnosticsIcon } from '@percona/peak-ui';
import type { ColorMode } from '@pmm/shared';
import { PreviewChip } from 'components/preview-chip';
import { createElement } from 'react';
import type { CombinedSettings } from 'contexts/settings';
import {
  EXTENSIONS_ATW_PATH,
  OM_PATH,
  PMM_NEW_NAV_GRAFANA_PATH,
  PMM_NEW_NAV_PATH,
} from 'lib/constants';
import type { DashboardFolder } from 'types/folders.types';
import type { HAInfo } from 'types/ha.types';
import type { NavItem } from 'types/navigation.types';
import { ServiceType } from 'types/services.types';
import type { GetUpdatesResponse, UpdateStatus } from 'types/updates.types';
import type { User } from 'types/user.types';
import {
  NAV_BACKUPS,
  NAV_EXPLORE,
  NAV_EXTENSIONS_MYSQL_BACKUPS,
  NAV_FOLDER_MAP,
  NAV_HELP,
  NAV_INVENTORY,
  NAV_MONGO,
  NAV_MYSQL,
  NAV_OS,
  NAV_OTHER_DASHBOARDS_TEMPLATE,
  NAV_POSTGRESQL,
  NAV_QAN,
  NAV_SIGN_IN,
  NAV_VALKEY,
  section,
} from './navigation.constants';
import {
  addAccount,
  addAdvisors,
  addAlerting,
  addConfiguration,
  addDashboards,
  addHighAvailability,
} from './navigation.utils';

// The sidebar, read top to bottom (see PMM-15353).

export const NAV_SECTIONS = {
  technologies: section('technologies', 'Technologies'),
  analysis: section('analysis', 'Analysis and alerting'),
  browse: section('browse', 'Browse'),
  administration: section('administration', 'Administration'),
};

// A copy of the technology's menu with its Grafana folder of other dashboards
// appended, so the shared constant never changes.
const technology = (item: NavItem, folders: DashboardFolder[]): NavItem => {
  const folder = folders.find((f) => f.title === NAV_FOLDER_MAP[item.id]);

  if (!folder) {
    return item;
  }

  return {
    ...item,
    children: [
      ...(item.children || []),
      {
        ...NAV_OTHER_DASHBOARDS_TEMPLATE,
        id: `${item.id}-other-dashboards`,
        url: `${PMM_NEW_NAV_GRAFANA_PATH}/dashboards/f/${folder.uid}/${item.id}`,
      },
    ],
  };
};

// Listed only while PMM monitors a service of that type; the OS is always there.
const TECHNOLOGIES: [ServiceType | undefined, NavItem][] = [
  [ServiceType.mongodb, NAV_MONGO],
  [ServiceType.mysql, NAV_MYSQL],
  [ServiceType.posgresql, NAV_POSTGRESQL],
  [ServiceType.valkey, NAV_VALKEY],
  [undefined, NAV_OS],
];

// Stored query metrics and live query sessions are separate jobs, so each is
// its own entry rather than a tab inside one Query analytics page.
export const NAV_REALTIME_QUERIES: NavItem = {
  id: 'realtime-queries',
  icon: TrackChanges,
  text: 'Real-time queries',
  url: `${PMM_NEW_NAV_PATH}/rta`,
  matches: ['*'],
};

// PMM Extensions apps are a technical preview.
const TECH_PREVIEW: Pick<NavItem, 'badge' | 'badgeAlwaysVisible'> = {
  badge: createElement(PreviewChip, { level: 'tech' }),
  badgeAlwaysVisible: true,
};

// The ATW app runs diagnostic checks against monitored databases; sending the
// results to a Percona Support case is one action inside it, not its purpose.
export const NAV_DIAGNOSTICS: NavItem = {
  id: 'extensions-atw',
  icon: SupportDiagnosticsIcon,
  text: 'Diagnostics',
  url: EXTENSIONS_ATW_PATH,
  matches: ['*'],
  ...TECH_PREVIEW,
};

// Its own tabs carry Fleet, Nodes, Automations and Settings, so it needs one
// entry, highlighted on all of them.
export const NAV_OM: NavItem = {
  id: 'app-operations-for-mongodb',
  text: 'Operations for MongoDB',
  url: OM_PATH,
  matches: ['*'],
  badge: createElement(PreviewChip, { level: 'developer' }),
  badgeAlwaysVisible: true,
};

export const NAV_APPS: NavItem = {
  id: 'apps',
  icon: AppsRounded,
  text: 'Apps',
};

export interface NavTreeInput {
  user?: User;
  isLoggedIn: boolean;
  haInfo?: HAInfo;
  settings?: CombinedSettings;
  serviceTypes: ServiceType[];
  folders: DashboardFolder[];
  updateStatus: UpdateStatus;
  versionInfo?: GetUpdatesResponse;
  colorMode: ColorMode;
  toggleColorMode: () => void;
}

// A section label is listed only above at least one entry.
const withSection = (heading: NavItem, items: NavItem[]): NavItem[] =>
  items.length ? [heading, ...items] : [];

export const buildNavTree = ({
  user,
  isLoggedIn,
  haInfo,
  settings,
  serviceTypes,
  folders,
  updateStatus,
  versionInfo,
  colorMode,
  toggleColorMode,
}: NavTreeInput): NavItem[] => {
  const extensions = !!user && !!settings?.extensionsEnabled;

  const technologies = TECHNOLOGIES.filter(
    ([type]) => !type || serviceTypes.includes(type)
  ).map(([, item]) => technology(item, folders));

  const analysis: NavItem[] = [];

  // Entries every signed-in user has come first, so they keep their place
  // whatever the role or switches.
  if (user) {
    analysis.push(
      NAV_QAN,
      NAV_REALTIME_QUERIES,
      addAlerting(
        settings?.alertingEnabled,
        settings?.frontend.unifiedAlertingEnabled,
        user
      )
    );
  }

  if (user?.isEditor && settings?.advisorEnabled) {
    analysis.push(addAdvisors());
  }

  if (user?.isEditor && settings?.frontend.exploreEnabled) {
    analysis.push(NAV_EXPLORE);
  }

  if (extensions) {
    analysis.push(NAV_DIAGNOSTICS);
  }

  // OM is served by pmm-managed, so it is not gated with PMM Extensions: any
  // signed-in user while the switch is on. Actions inside are admin-only.
  const apps: NavItem[] = [];

  if (user && settings?.omEnabled) {
    apps.push(NAV_OM);
  }

  if (extensions) {
    apps.push({ ...NAV_EXTENSIONS_MYSQL_BACKUPS, ...TECH_PREVIEW });
  }

  if (user?.isPMMAdmin && settings?.backupManagementEnabled) {
    apps.push(...NAV_BACKUPS);
  }

  const browse: NavItem[] = [addDashboards(serviceTypes, user)];

  if (apps.length) {
    browse.push({ ...NAV_APPS, children: apps });
  }

  if (user?.isPMMAdmin) {
    browse.push(NAV_INVENTORY);
  }

  const administration: NavItem[] = [];

  if (haInfo?.enabled) {
    administration.push(addHighAvailability(haInfo));
  }

  if (user?.isPMMAdmin) {
    administration.push(addConfiguration(updateStatus, versionInfo, settings));
  }

  const items: NavItem[] = [
    ...withSection(NAV_SECTIONS.technologies, technologies),
    ...withSection(NAV_SECTIONS.analysis, analysis),
    ...withSection(NAV_SECTIONS.browse, browse),
    ...withSection(NAV_SECTIONS.administration, administration),
  ];

  if (user) {
    if (isLoggedIn) {
      items.push(addAccount(user, colorMode, toggleColorMode, settings));
    }

    items.push(NAV_HELP);
  }

  if (!isLoggedIn) {
    items.push(NAV_SIGN_IN);
  }

  return items;
};
