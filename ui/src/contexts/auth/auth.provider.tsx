import { FC, PropsWithChildren, useMemo } from 'react';
import { AuthContext } from './auth.context';
import { useQuery } from '@tanstack/react-query';
import { rotateToken } from 'api/auth';
import { getRefetchInterval, redirectToLogin } from './auth.utils';
import { AxiosError, HttpStatusCode } from 'axios';

export const AuthProvider: FC<PropsWithChildren> = ({ children }) => {
  const { error, isLoading } = useQuery({
    queryKey: ['rotateToken'],
    queryFn: () => rotateToken(),
    refetchInterval: () => getRefetchInterval(),
    refetchIntervalInBackground: true,
    retry: false,
  });
  const shouldRedirectToLogin = useMemo(() => {
    const response = (error as AxiosError)?.response;
    return (
      response?.status === HttpStatusCode.Unauthorized ||
      response?.status === HttpStatusCode.InternalServerError
    );
  }, [error]);

  if (shouldRedirectToLogin) {
    redirectToLogin();
    return null;
  }

  if (isLoading) {
    return null;
  }

  return (
    <AuthContext.Provider value={{ isLoading }}>
      {children}
    </AuthContext.Provider>
  );
};
