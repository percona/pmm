import { AxiosError } from 'axios';
import {
  OrgRole,
  User,
  GetUserResponse,
  UserOrg,
  UserInfo,
  UserPreferences,
} from 'types/user.types';

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
