import DarkModeOutlined from '@mui/icons-material/DarkModeOutlined';
import LightModeOutlined from '@mui/icons-material/LightModeOutlined';
import { NavItem } from 'types/navigation.types';
import { User, UserPreferences } from 'types/user.types';
import { PMM_NEW_NAV_GRAFANA_PATH } from 'lib/constants';
import { ColorMode } from '@pmm/shared';
import {
  NAV_ACCOUNT,
  NAV_CHANGE_PASSWORD,
  NAV_CONFIGURATION,
  NAV_HIGH_AVAILABILITY,
  NAV_HOME_PAGE,
  NAV_SIGN_OUT,
  NAV_THEME_TOGGLE,
} from './navigation.constants';
import { CombinedSettings } from 'contexts/settings';
import { GetUpdatesResponse, UpdateStatus } from 'types/updates.types';
import { HighAvailabilityIcon } from 'components/ha-icon';
import { HighAvailabilityBadge } from 'components/ha-badge';
import { HAInfo } from 'types/ha.types';

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
  versionInfo?: GetUpdatesResponse
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

  return NAV_CONFIGURATION;
};

// A single entry: the namespace rides on the link, and the plain dashboard path
// in `matches` keeps the entry highlighted however the dashboard was reached.
export const addHighAvailability = ({ health, namespace }: HAInfo): NavItem => {
  const { url = '' } = NAV_HIGH_AVAILABILITY;
  const item = { ...NAV_HIGH_AVAILABILITY, url, matches: [url] };

  if (namespace) {
    item.url = `${url}?var-namespace=${encodeURIComponent(namespace)}`;
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
