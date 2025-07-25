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
  orgId: number;
  orgRole: OrgRole | '';
  isAuthorized: boolean;
  isViewer: boolean;
  isEditor: boolean;
  isPMMAdmin: boolean;
  orgs: UserOrg[];
}

// comes from grafana
export interface GetUserResponse {
  id: number;
  email: string;
  name: string;
  login: string;
  createdAt: string;
  orgId: number;
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

export interface UpdatePreferencesBody {
  theme: ColorMode;
}
