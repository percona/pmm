import DarkModeOutlined from '@mui/icons-material/DarkModeOutlined';
import LightModeOutlined from '@mui/icons-material/LightModeOutlined';
import { NavItem } from 'types/navigation.types';
import { User } from 'types/user.types';
import { ServiceType } from 'types/services.types';
import { ColorMode } from '@pmm/shared';
import {
  NAV_ACCOUNT,
  NAV_ADVISORS,
  NAV_ADVISORS_CHECKS,
  NAV_ADVISORS_INSIGHTS,
  NAV_ADVISORS_RUNS,
  NAV_ADVISORS_SCHEDULER,
  NAV_ALERTS_RULES,
  NAV_ALERTS,
  NAV_ALERTS_CONTACT_POINTS,
  NAV_ALERTS_NOTIFICATION_POLICIES,
  NAV_ALERTS_SETTINGS,
  NAV_ALERTS_TEMPLATES,
  NAV_ALERTS_STATUS,
  NAV_CHANGE_PASSWORD,
  NAV_CONFIGURATION,
  NAV_DASHBOARDS,
  NAV_DASHBOARDS_BROWSE,
  NAV_DASHBOARDS_LIBRARY_PANELS,
  NAV_DASHBOARDS_PLAYLISTS,
  NAV_DASHBOARDS_SHARED,
  NAV_DASHBOARDS_SNAPSHOTS,
  NAV_HAPROXY,
  NAV_PROXYSQL,
  NAV_SIGN_OUT,
  NAV_THEME_TOGGLE,
  NAV_ALERTS_SILENCES,
  NAV_ALERTS_GROUPS,
  NAV_HIGH_AVAILABILITY,
  NAV_ACCESS_CONTROL,
} from './navigation.constants';
import { CombinedSettings } from 'contexts/settings';
import { GetUpdatesResponse, UpdateStatus } from 'types/updates.types';
import { HighAvailabilityIcon } from 'components/ha-icon';
import { HighAvailabilityBadge } from 'components/ha-badge';
import { HAInfo } from 'types/ha.types';

export const addAlerting = (
  alertingEnabled = false,
  unifiedAlertingEnabled = false,
  user?: User
): NavItem => {
  const children: NavItem[] = [NAV_ALERTS_RULES];

  if (alertingEnabled) {
    children.push(NAV_ALERTS_STATUS);

    if (user?.isEditor) {
      children.push(NAV_ALERTS_TEMPLATES);
    }
  }

  if (user) {
    children.push(NAV_ALERTS_SILENCES);
    children.push(NAV_ALERTS_GROUPS);

    if (unifiedAlertingEnabled && user.isPMMAdmin) {
      children.push(NAV_ALERTS_SETTINGS);
    }

    if (unifiedAlertingEnabled) {
      children.push(NAV_ALERTS_CONTACT_POINTS);
      children.push(NAV_ALERTS_NOTIFICATION_POLICIES);
    }
  }

  return { ...NAV_ALERTS, children };
};

export const addDashboards = (
  serviceTypes: ServiceType[],
  user?: User
): NavItem => {
  const children: NavItem[] = [NAV_DASHBOARDS_BROWSE];

  if (serviceTypes.includes(ServiceType.proxysql)) {
    children.push(NAV_PROXYSQL);
  }

  if (serviceTypes.includes(ServiceType.haproxy)) {
    children.push(NAV_HAPROXY);
  }

  if (user) {
    children.push(
      NAV_DASHBOARDS_SHARED,
      NAV_DASHBOARDS_PLAYLISTS,
      NAV_DASHBOARDS_SNAPSHOTS,
      NAV_DASHBOARDS_LIBRARY_PANELS
    );
  }

  return { ...NAV_DASHBOARDS, children };
};

export const addAdvisors = (): NavItem => ({
  ...NAV_ADVISORS,
  children: [
    NAV_ADVISORS_INSIGHTS,
    NAV_ADVISORS_CHECKS,
    NAV_ADVISORS_RUNS,
    NAV_ADVISORS_SCHEDULER,
  ],
});

export const addAccount = (
  user: User,
  colorMode: ColorMode,
  toggleMode: () => void,
  settings?: CombinedSettings
): NavItem => {
  const name = (user.name || '').split(' ')[0];
  const children = [...(NAV_ACCOUNT.children || [])];
  const targetMode = colorMode === 'light' ? 'dark' : 'light';

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
    icon: colorMode === 'light' ? DarkModeOutlined : LightModeOutlined,
    text: `Switch to ${targetMode} mode`,
    onClick: toggleMode,
  });

  children.push(NAV_SIGN_OUT);

  return {
    ...NAV_ACCOUNT,
    children,
    text: NAV_ACCOUNT.text + (name ? `: ${name}` : ''),
  };
};

export const addConfiguration = (
  status: UpdateStatus,
  versionInfo?: GetUpdatesResponse,
  settings?: CombinedSettings
): NavItem => {
  const { updateAvailable, installed, latest } = versionInfo || {};
  // Without a dev build's suffix, which would not fit the menu.
  const version = installed?.version?.split('-')[0];
  const children = (NAV_CONFIGURATION.children || []).map((child) => {
    if (child.id !== 'updates') {
      return child;
    }

    return {
      ...child,
      secondaryText: updateAvailable
        ? `v${version} → v${latest?.version}`
        : status === UpdateStatus.UpdateClients
          ? 'Clients out of date'
          : version
            ? `v${version}, up to date`
            : undefined,
      badge:
        status === UpdateStatus.Pending || status === UpdateStatus.UpdateClients
          ? { label: 'New' }
          : undefined,
    };
  });

  if (settings?.enableAccessControl) {
    children.push(NAV_ACCESS_CONTROL);
  }

  return { ...NAV_CONFIGURATION, children };
};

export const addHighAvailability = ({ health, namespace }: HAInfo): NavItem => {
  const item = { ...NAV_HIGH_AVAILABILITY };

  if (namespace) {
    item.url = `${item.url}?var-namespace=${encodeURIComponent(namespace)}`;
  }

  item.badge = <HighAvailabilityBadge health={health} />;
  item.icon = <HighAvailabilityIcon health={health} />;
  item.badgeAlwaysVisible = true;

  return item;
};
