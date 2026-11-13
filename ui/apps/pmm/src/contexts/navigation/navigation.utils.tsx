import DarkModeOutlined from '@mui/icons-material/DarkModeOutlined';
import LightModeOutlined from '@mui/icons-material/LightModeOutlined';
import { NavItem } from 'types/navigation.types';
import { User, UserPreferences } from 'types/user.types';
import { PMM_NEW_NAV_GRAFANA_PATH } from 'lib/constants';
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
  NAV_SIGN_OUT,
  NAV_THEME_TOGGLE,
  NAV_ALERTS_SILENCES,
  NAV_ALERTS_GROUPS,
  NAV_HIGH_AVAILABILITY,
  NAV_ACCESS_CONTROL,
  NAV_HOME_PAGE,
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

    if (!user.isAnonymous) {
      children.push(NAV_ALERTS_GROUPS);

      if (unifiedAlertingEnabled && user.isPMMAdmin) {
        children.push(NAV_ALERTS_SETTINGS);
      }
    }

    if (unifiedAlertingEnabled) {
      children.push(NAV_ALERTS_CONTACT_POINTS);
      children.push(NAV_ALERTS_NOTIFICATION_POLICIES);
    }
  }

  return { ...NAV_ALERTS, children };
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
  const updates = NAV_CONFIGURATION.children?.find((c) => c.id === 'updates');
  const { updateAvailable, installed, latest } = versionInfo || {};

  if (!updates) {
    return NAV_CONFIGURATION;
  }

  if (updateAvailable) {
    updates.secondaryText = `Update from v${installed?.version?.slice(0, 5)} to v${latest?.version}`;
  } else if (installed?.version) {
    updates.secondaryText = `Current: v${installed?.version} (up to date)`;
  }

  if (
    status === UpdateStatus.Pending ||
    status === UpdateStatus.UpdateClients
  ) {
    updates.badge = {
      label: 'New',
    };
  } else {
    updates.badge = undefined;
  }

  if (settings?.enableAccessControl) {
    return {
      ...NAV_CONFIGURATION,
      children: [...(NAV_CONFIGURATION.children || []), NAV_ACCESS_CONTROL],
    };
  }

  return NAV_CONFIGURATION;
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

export const addHomePage = (preferences?: UserPreferences): NavItem => {
  if (preferences?.homeDashboardUID) {
    return {
      ...NAV_HOME_PAGE,
      // highlight also the custom home dashboard
      matches: [
        ...(NAV_HOME_PAGE.matches || []),
        `${PMM_NEW_NAV_GRAFANA_PATH}/d/${preferences.homeDashboardUID}`,
      ],
    };
  }

  return NAV_HOME_PAGE;
};
