import { FC, PropsWithChildren, useMemo } from 'react';
import { UserContext } from './user.context';
import {
  useCurrentUser,
  useCurrentUserOrgs,
  useUserInfo,
} from 'hooks/api/useUser';
import { getPerconaUser, isAuthorized } from './user.utils';
import { useAuth } from 'contexts/auth';

export const UserProvider: FC<PropsWithChildren> = ({ children }) => {
  const auth = useAuth();
  const userQuery = useCurrentUser({
    enabled: auth.isLoggedIn,
  });
  const userInfoQuery = useUserInfo({
    enabled: auth.isLoggedIn,
  });
  const orgsQuery = useCurrentUserOrgs({
    enabled: auth.isLoggedIn,
  });
  const user = useMemo(() => {
    if (!userQuery.data || !orgsQuery.data || !userInfoQuery.data) {
      return;
    }

    return getPerconaUser(
      userQuery.data,
      orgsQuery.data,
      userInfoQuery.data,
      isAuthorized(userQuery.error)
    );
  }, [userQuery, orgsQuery, userInfoQuery]);

  return (
    <UserContext.Provider
      value={{
        isLoading: userQuery.isLoading || orgsQuery.isLoading,
        user,
      }}
    >
      {children}
    </UserContext.Provider>
  );
};
