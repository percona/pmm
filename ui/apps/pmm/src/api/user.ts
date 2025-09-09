import {
  GetUserResponse,
  UpdatePreferencesBody,
  UserOrg,
} from 'types/user.types';
import { grafanaApi } from './api';

export const getCurrentUser = async () => {
  const res = await grafanaApi.get<GetUserResponse>('/user');
  return res.data;
};

export const getCurrentUserOrgs = async () => {
  const res = await grafanaApi.get<UserOrg[]>('/user/orgs');
  return res.data;
};

export const updatePreferences = async (
  preferences: Partial<UpdatePreferencesBody>
) => {
  const res = await grafanaApi.patch('/user/preferences', preferences);
  return res.data;
};
