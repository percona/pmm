import type { QueryClient } from '@tanstack/react-query';
import { clearClientSession } from './auth.clientSession';

const ROTATE_TOKEN_QUERY_KEY = ['rotateToken'] as const;

/** Grafana logged the user out (e.g. password change); sync PMM shell auth state. */
export const handleGrafanaUserLoggedOut = (queryClient: QueryClient) => {
  clearClientSession();
  queryClient.setQueryData(ROTATE_TOKEN_QUERY_KEY, undefined);
  void queryClient.invalidateQueries({ queryKey: ROTATE_TOKEN_QUERY_KEY });
};
