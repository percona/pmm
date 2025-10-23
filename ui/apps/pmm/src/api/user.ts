import {
  GetUserResponse,
  UpdatePreferencesBody,
  UpdateUserInfoPayload,
  UserInfo,
  UserOrg,
} from 'types/user.types';
import { api, grafanaApi } from './api';

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

export const getUserInfo = async () => {
  const res = await api.get<UserInfo>('/users/me');
  return res.data;
};

export const updateUserInfo = async (payload: UpdateUserInfoPayload) => {
  const res = await api.put<UserInfo>('/users/me', payload);
  return res.data;
};
