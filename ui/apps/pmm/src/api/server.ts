import { GetServerVersionResponse } from 'types/server.types';
import { api } from './api';

export const getServerVersion = async () => {
  const res = await api.get<GetServerVersionResponse>('/server/version', {
    // The version is polled while the server is restarting during an upgrade,
    // so the failures it produces must not reach the error snackbar.
    disableNotifications: true,
  });
  return res.data;
};
