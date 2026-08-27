import { FC, PropsWithChildren, useEffect, useMemo } from 'react';
import { AuthContext } from './auth.context';
import { useQuery } from '@tanstack/react-query';
import { rotateToken } from 'api/auth';
import {
  establishClientSession,
  ensureClientSessionListener,
  useClientSession,
} from './auth.clientSession';
import { ROTATE_TOKEN_QUERY_KEY } from './auth.queryKeys';
import { getRefetchInterval, redirectToLogin } from './auth.utils';
import { AxiosError, HttpStatusCode } from 'axios';
import { useFrontendSettings } from 'hooks/api/useSettings';

export const AuthProvider: FC<PropsWithChildren> = ({ children }) => {
  const settings = useFrontendSettings({ retry: false });
  const clientSessionEstablished = useClientSession();

  useEffect(() => {
    ensureClientSessionListener();
  }, []);

  const { error, isLoading, data } = useQuery({
    queryKey: ROTATE_TOKEN_QUERY_KEY,
    queryFn: async () => {
      const token = await rotateToken();
      establishClientSession();
      return token;
    },
    refetchInterval: () => getRefetchInterval(),
    refetchIntervalInBackground: true,
    retry: false,
  });

  const hasServerSession = Boolean(data);
  const isLoggedIn = hasServerSession && clientSessionEstablished;

  const shouldRedirectToLogin = useMemo(() => {
    if (settings.data?.anonymousEnabled) {
      return false;
    }

    const response = (error as AxiosError)?.response;
    if (
      response?.status === HttpStatusCode.Unauthorized ||
      response?.status === HttpStatusCode.InternalServerError
    ) {
      return true;
    }

    return hasServerSession && !clientSessionEstablished;
  }, [
    clientSessionEstablished,
    error,
    hasServerSession,
    settings.data?.anonymousEnabled,
  ]);

  if (isLoading || settings.isLoading) {
    return null;
  }

  if (shouldRedirectToLogin) {
    redirectToLogin();
    return null;
  }

  return (
    <AuthContext.Provider value={{ isLoading, isLoggedIn }}>
      {children}
    </AuthContext.Provider>
  );
};
