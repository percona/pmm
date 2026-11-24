import type { ColorMode } from '@pmm/shared';
import type { CombinedSettings } from 'contexts/settings';
import { PMM_NEW_NAV_GRAFANA_PATH } from 'lib/constants';
import type { DashboardFolder } from 'types/folders.types';
import type { HAInfo } from 'types/ha.types';
import type { NavItem } from 'types/navigation.types';
import { ServiceType } from 'types/services.types';
import type { GetUpdatesResponse, UpdateStatus } from 'types/updates.types';
import type { User } from 'types/user.types';
import {
  METRICS_DRILLDOWN_APP,
  NAV_APPS,
  NAV_BACKUPS,
  NAV_DIAGNOSTICS,
  NAV_EXPLORE,
  NAV_EXPLORE_METRICS,
  NAV_EXTENSIONS_MYSQL_BACKUPS,
  NAV_FOLDER_MAP,
  NAV_HELP,
  NAV_INVENTORY,
  NAV_MONGO,
  NAV_MYSQL,
  NAV_OM,
  NAV_OS,
  NAV_OTHER_DASHBOARDS_TEMPLATE,
  NAV_POSTGRESQL,
  NAV_QAN,
  NAV_REALTIME_QUERIES,
  NAV_SECTIONS,
  NAV_SIGN_IN,
  NAV_VALKEY,
} from './navigation.constants';
import {
  addAccount,
  addAdvisors,
  addAlerting,
  addConfiguration,
  addDashboards,
  addHighAvailability,
} from './navigation.utils';

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

// The sidebar, read top to bottom (see PMM-15353).
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

    if (METRICS_DRILLDOWN_APP in settings.frontend.apps) {
      analysis.push(NAV_EXPLORE_METRICS);
    }
  }

  if (extensions) {
    analysis.push(NAV_DIAGNOSTICS);
  }

  const apps: NavItem[] = [];

  // First, as it has no heading: after another app's entries it would read
  // as one of them.
  if (extensions) {
    apps.push(NAV_EXTENSIONS_MYSQL_BACKUPS);
  }

  // OM is served by pmm-managed, so it is not gated with PMM Extensions: any
  // signed-in user while the switch is on. Actions inside are admin-only.
  if (user && settings?.omEnabled) {
    apps.push(...NAV_OM);
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
