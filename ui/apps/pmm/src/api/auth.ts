import { AxiosResponse } from 'axios';
import { grafanaApi } from './api';

export const rotateToken = async (): Promise<AxiosResponse['data']> => {
  const res = await grafanaApi.post('/user/auth-tokens/rotate');
  return res.data;
};
