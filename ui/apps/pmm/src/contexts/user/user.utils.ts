import { AxiosError } from 'axios';
import { OrgRole, User, GetUserResponse } from 'types/user.types';

export const getPerconaUser = (
  user: GetUserResponse,
  isAuthorized: boolean
): User => ({
  id: user.id,
  isPMMAdmin: isPMMAdmin(user),
  isAuthorized,
  orgRole: user.orgRole,
});

export const isAuthorized = (error?: Error | null) =>
  !error || (error as AxiosError).response?.status !== 401;

export const isPMMAdmin = (user: GetUserResponse): boolean =>
  user.isGrafanaAdmin || user.orgRole === OrgRole.Admin;
