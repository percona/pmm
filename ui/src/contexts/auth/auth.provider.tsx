import { FC, PropsWithChildren, useEffect } from 'react';
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

  useEffect(() => {
    const response = (error as AxiosError)?.response;

    if (response?.status === HttpStatusCode.Unauthorized) {
      redirectToLogin();
    }
  }, [error]);

  return (
    <AuthContext.Provider value={{ isLoading }}>
      {children}
    </AuthContext.Provider>
  );
};
