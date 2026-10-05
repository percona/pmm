import { AxiosResponse } from 'axios';
import { grafanaApi } from './api';

export const ROTATE_TOKEN_URL = '/user/auth-tokens/rotate';

let rotation: Promise<AxiosResponse> | null = null;

/**
 * Grafana uses client-side session token rotation: `token_rotation_interval_minutes` is a
 * hard deadline, not a refresh hint, and every request past it is rejected until the client
 * rotates. Rotation is driven from two places — the scheduled refresh in AuthProvider and
 * the 401 interceptor — so it is single-flighted to keep them from rotating twice in a row,
 * which would strand any request still carrying the older token.
 */
export const rotateToken = async (): Promise<AxiosResponse['data']> => {
  rotation ??= grafanaApi.post(ROTATE_TOKEN_URL).finally(() => {
    rotation = null;
  });

  const response = await rotation;

  return response.data;
};
