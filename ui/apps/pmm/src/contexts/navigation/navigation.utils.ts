import { NavItem } from 'lib/types';
import { ServiceType } from 'types/services.types';
import { User } from 'types/user.types';
import { FrontendSettings, Settings } from 'types/settings.types';
import { Advisor } from 'types/advisors.types';
import { groupAdvisorsIntoCategories } from 'lib/utils/advisors.utils';
import { PMM_NEW_NAV_GRAFANA_PATH } from 'lib/constants';
import { ColorMode } from '@pmm/shared';
import {
  NAV_ACCOUNT,
  NAV_ADVISORS,
  NAV_ADVISORS_INSIGHTS,
  NAV_ALERTS,
  NAV_ALERTS_CONTACT_POINTS,
  NAV_ALERTS_FIRED,
  NAV_ALERTS_NOTIFICATION_POLICIES,
  NAV_ALERTS_SETTINGS,
  NAV_ALERTS_TEMPLATES,
  NAV_CHANGE_PASSWORD,
  NAV_DASHBOARDS,
  NAV_DASHBOARDS_BROWSE,
  NAV_DASHBOARDS_LIBRARY_PANELS,
  NAV_DASHBOARDS_PLAYLISTS,
  NAV_DASHBOARDS_SHARED,
  NAV_DASHBOARDS_SNAPSHOTS,
  NAV_EXPLORE,
  NAV_EXPLORE_BUILDER,
  NAV_EXPLORE_METRICS,
  NAV_FOLDER_MAP,
  NAV_HAPROXY,
  NAV_MONGO,
  NAV_MYSQL,
  NAV_OS,
  NAV_OTHER_DASHBOARDS_TEMPLATE,
  NAV_POSTGRESQL,
  NAV_PROXYSQL,
  NAV_SIGN_OUT,
  NAV_THEME_TOGGLE,
} from './navigation.constants';
import { CombinedSettings } from 'contexts/settings';
import { capitalize } from 'utils/textUtils';
import { DashboardFolder } from 'types/folders.types';

export const addOtherDashboardsItem = (
  rootNode: NavItem,
  folders: DashboardFolder[]
) => {
  const id = rootNode.id + '-other-dashboards';
  const folder = folders.find(
    (f) => rootNode.id && NAV_FOLDER_MAP[rootNode.id] === f.title
  );
  const exists = rootNode.children?.some((i) => i.id === id);

  if (folder && !exists) {
    rootNode.children?.push({
      ...NAV_OTHER_DASHBOARDS_TEMPLATE,
      id,
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/dashboards/f/${folder.uid}/${rootNode.id}`,
    });
  }
};

export const addAllDashboardItem = (
  types: ServiceType[],
  user?: User
): NavItem => {
  const children: NavItem[] = [NAV_DASHBOARDS_BROWSE];

  if (types.includes(ServiceType.proxysql)) {
    children.push(NAV_PROXYSQL);
  }

  if (types.includes(ServiceType.haproxy)) {
    children.push(NAV_HAPROXY);
  }

  if (user) {
    children.push(NAV_DASHBOARDS_SHARED);
    children.push(NAV_DASHBOARDS_PLAYLISTS);
    children.push(NAV_DASHBOARDS_SNAPSHOTS);
    children.push(NAV_DASHBOARDS_LIBRARY_PANELS);
  }

  return { ...NAV_DASHBOARDS, children };
};

export const addDashboardItems = (
  types: ServiceType[],
  folders: DashboardFolder[],
  user?: User
): NavItem[] => {
  const children: NavItem[] = [];

  if (types.includes(ServiceType.mysql)) {
    addOtherDashboardsItem(NAV_MYSQL, folders);
    children.push(NAV_MYSQL);
  }

  if (types.includes(ServiceType.mongodb)) {
    addOtherDashboardsItem(NAV_MONGO, folders);
    children.push(NAV_MONGO);
  }

  if (types.includes(ServiceType.posgresql)) {
    addOtherDashboardsItem(NAV_POSTGRESQL, folders);
    children.push(NAV_POSTGRESQL);
  }

  addOtherDashboardsItem(NAV_OS, folders);
  children.push(NAV_OS);

  children.push(addAllDashboardItem(types, user));

  return children;
};

export const addAlerting = (enabled = false, user?: User): NavItem => {
  const children: NavItem[] = [];

  if (enabled) {
    children.push(NAV_ALERTS_FIRED);
  }

  children.push(NAV_ALERTS_CONTACT_POINTS);
  children.push(NAV_ALERTS_NOTIFICATION_POLICIES);
  children.push(NAV_ALERTS_SETTINGS);

  if (enabled && user?.isEditor) {
    children.push(NAV_ALERTS_TEMPLATES);
  }

  return { ...NAV_ALERTS, children };
};

export const addExplore = (frontendSettings: FrontendSettings): NavItem => {
  const children: NavItem[] = [NAV_EXPLORE_BUILDER];

  if (frontendSettings.featureToggles.exploreMetrics) {
    children.push(NAV_EXPLORE_METRICS);
  }

  return { ...NAV_EXPLORE, children };
};

export const addAdvisors = (advisors: Advisor[]): NavItem => {
  const children: NavItem[] = [NAV_ADVISORS_INSIGHTS];
  const categories = groupAdvisorsIntoCategories(advisors);

  for (const category of Object.keys(categories)) {
    children.push({
      id: `advisors-${category}`,
      text: `${capitalize(category)} Advisors`,
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/advisors/${category}`,
    });
  }

  return { ...NAV_ADVISORS, children };
};

export const addAccount = (
  user: User,
  colorMode: ColorMode,
  toggleMode: () => void,
  settings?: CombinedSettings
): NavItem => {
  const name = (user.name || '').split(' ')[0];
  const children = [...(NAV_ACCOUNT.children || [])];
  const targetTheme = colorMode === 'light' ? 'Dark' : 'Light';

  if (
    !(
      settings?.frontend.disableLoginForm ||
      settings?.frontend.auth.disableLogin
    )
  ) {
    children.push(NAV_CHANGE_PASSWORD);
  }

  children.push({
    ...NAV_THEME_TOGGLE,
    icon: colorMode === 'light' ? 'theme-dark' : 'theme-light',
    text: `Change to ${targetTheme} Theme`,
    onClick: toggleMode,
  });

  children.push(NAV_SIGN_OUT);

  return {
    ...NAV_ACCOUNT,
    children,
    text: NAV_ACCOUNT.text + (name ? `: ${name}` : ''),
  };
};
