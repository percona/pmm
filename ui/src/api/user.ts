import { GetUserResponse } from 'types/user.types';
import { grafanaApi } from './api';

export const getCurrentUser = async () => {
  const res = await grafanaApi.get<GetUserResponse>('/user');
  return res.data;
};
