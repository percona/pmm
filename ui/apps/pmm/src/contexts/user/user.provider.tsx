import { FC, PropsWithChildren, useMemo } from 'react';
import { UserContext } from './user.context';
import { useCurrentUser } from 'hooks/api/useUser';
import { useSettings } from 'hooks/api/useSettings';
import { getPerconaUser, isAuthorized } from './user.utils';
import { useAuth } from 'contexts/auth';

export const UserProvider: FC<PropsWithChildren> = ({ children }) => {
  const auth = useAuth();
  const { data, isLoading: isLoadingUser } = useCurrentUser({
    enabled: auth.isLoggedIn,
  });
  const { error, isLoading: isLoadingSettings } = useSettings({
    retry: false,
    enabled: auth.isLoggedIn,
  });
  const user = useMemo(
    () => data && getPerconaUser(data, isAuthorized(error)),
    [data, error]
  );

  return (
    <UserContext.Provider
      value={{
        isLoading: isLoadingSettings || isLoadingUser,
        user,
      }}
    >
      {children}
    </UserContext.Provider>
  );
};
