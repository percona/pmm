import { ColorMode } from '@pmm/shared';

// comes from grafana
export enum OrgRole {
  None = 'None',
  Viewer = 'Viewer',
  Editor = 'Editor',
  Admin = 'Admin',
}

export interface User {
  id: number;
  name: string;
  login: string;
  isAnonymous: boolean;
  orgId: number;
  orgRole: OrgRole | '';
  isAuthorized: boolean;
  isViewer: boolean;
  isEditor: boolean;
  isPMMAdmin: boolean;
  orgs: UserOrg[];
  info: UserInfo;
  preferences: UserPreferences;
}

// comes from grafana
export interface GetUserResponse {
  id: number;
  email: string;
  name: string;
  login: string;
  createdAt: string;
  orgId: number;
  isAnonymous: boolean;
  isDisabled: boolean;
  isExternal: boolean;
  isExtarnallySynced: boolean;
  isGrafanaAdmin: boolean;
  isGrafanaAdminExternallySynced: boolean;
  theme: 'dark' | 'light' | 'system' | '';
}

export interface UserOrg {
  orgId: number;
  name: string;
  role: OrgRole;
}

export interface GetPreferenceResponse {
  theme?: ColorMode;
  homeDashboardUID?: string;
  timezone?: string;
}

export interface UpdatePreferencesBody {
  theme?: ColorMode;
  homeDashboardUID?: string;
  timezone?: string;
}

export type UserPreferences = GetPreferenceResponse;

export interface UserInfo {
  userId: number;
  alertingTourCompleted: boolean;
  productTourCompleted: boolean;
  snoozedAt: string | null;
  snoozeCount: number;
  snoozedPmmVersion: string;
}

export type UpdateUserInfoPayload = Partial<
  Omit<UserInfo, 'userId' | 'snoozeCount' | 'snoozedAt'>
>;
