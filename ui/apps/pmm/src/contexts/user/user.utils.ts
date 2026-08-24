import { AxiosError } from 'axios';
import {
  OrgRole,
  User,
  GetUserResponse,
  UserOrg,
  UserInfo,
  UserPreferences,
} from 'types/user.types';

export const ANONYMOUS_USER_INFO: UserInfo = {
  userId: 0,
  alertingTourCompleted: false,
  productTourCompleted: false,
  snoozedAt: null,
  snoozeCount: 0,
  snoozedPmmVersion: '',
};

export const createAnonymousUser = (overrides: Partial<User> = {}): User => ({
  id: 0,
  login: 'anonymous',
  name: 'Anonymous',
  isAuthorized: true,
  isViewer: true,
  isEditor: false,
  isPMMAdmin: false,
  orgId: 1,
  orgRole: OrgRole.Viewer,
  orgs: [],
  preferences: {},
  info: ANONYMOUS_USER_INFO,
  ...overrides,
  isAnonymous: true,
});

export const getPerconaUser = (
  user: GetUserResponse,
  orgs: UserOrg[],
  info: UserInfo,
  preferences: UserPreferences,
  isAuthorized: boolean
): User => {
  const orgRole = orgs.find((org) => org.orgId === user.orgId)?.role || '';

  return {
    id: user.id,
    isAnonymous: user.isAnonymous,
    isAuthorized,
    name: user.name,
    login: user.login,
    orgs,
    orgRole,
    info,
    orgId: user.orgId,
    preferences,
    isViewer: orgRole === OrgRole.Viewer,
    isEditor: orgRole === OrgRole.Editor || orgRole === OrgRole.Admin,
    isPMMAdmin: user.isGrafanaAdmin || orgRole === OrgRole.Admin,
  };
};

export const isAuthorized = (error?: Error | null) =>
  !error || (error as AxiosError).response?.status !== 401;
