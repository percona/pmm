import { FC, PropsWithChildren, useMemo } from 'react';
import { AuthContext } from './auth.context';
import { useQuery } from '@tanstack/react-query';
import { rotateToken } from 'api/auth';
import { getRefetchInterval, redirectToLogin } from './auth.utils';
import { AxiosError, HttpStatusCode } from 'axios';
import { useFrontendSettings } from 'hooks/api/useSettings';

export const AuthProvider: FC<PropsWithChildren> = ({ children }) => {
  const settings = useFrontendSettings({
    retry: false,
  });
  const { error, isLoading } = useQuery({
    queryKey: ['rotateToken'],
    queryFn: () => rotateToken(),
    refetchInterval: () => getRefetchInterval(),
    refetchIntervalInBackground: true,
    retry: false,
  });
  const shouldRedirectToLogin = useMemo(() => {
    const response = (error as AxiosError)?.response;

    if (settings.data?.anonymousEnabled) {
      return false;
    }

    return (
      response?.status === HttpStatusCode.Unauthorized ||
      response?.status === HttpStatusCode.InternalServerError
    );
  }, [error, settings.data?.anonymousEnabled]);

  if (isLoading || settings.isLoading) {
    return null;
  }

  if (shouldRedirectToLogin) {
    redirectToLogin();
    return null;
  }

  return (
    <AuthContext.Provider value={{ isLoading }}>
      {children}
    </AuthContext.Provider>
  );
};
