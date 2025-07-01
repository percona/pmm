// comes from grafana
export enum OrgRole {
  None = 'None',
  Viewer = 'Viewer',
  Editor = 'Editor',
  Admin = 'Admin',
}

export interface User {
  id: number;
  orgRole: OrgRole | '';
  isPMMAdmin: boolean;
  isAuthorized: boolean;
}

// comes from grafana
export interface GetUserResponse {
  id: number;
  email: string;
  name: string;
  login: string;
  createdAt: string;
  orgRole: OrgRole;
  isDisabled: boolean;
  isExternal: boolean;
  isExtarnallySynced: boolean;
  isGrafanaAdmin: boolean;
  isGrafanaAdminExternallySynced: boolean;
  theme: 'dark' | 'light' | 'system' | '';
}
