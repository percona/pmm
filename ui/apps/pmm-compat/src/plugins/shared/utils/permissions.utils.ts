import { OrgRole, type CurrentUserDTO } from '@grafana/data';

export const isPmmAdmin = (user: CurrentUserDTO) =>
  user.isGrafanaAdmin || user.orgRole === OrgRole.Admin;

export const isEditor = (user: CurrentUserDTO) =>
  user.orgRole === OrgRole.Editor;
