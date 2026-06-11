import { FC, PropsWithChildren, useMemo } from 'react';
import { UserContext } from './user.context';
import {
  useCurrentUser,
  useCurrentUserOrgs,
  useUserInfo,
  useUserPreferences,
} from 'hooks/api/useUser';
import { getPerconaUser, isAuthorized } from './user.utils';
import { useAuth } from 'contexts/auth';
import { GetPreferenceResponse, UserInfo } from 'types/user.types';

export const UserProvider: FC<PropsWithChildren> = ({ children }) => {
  const auth = useAuth();
  const userQuery = useCurrentUser();
  const userInfoQuery = useUserInfo({
    enabled: auth.isLoggedIn,
  });
  const orgsQuery = useCurrentUserOrgs();
  const preferencesQuery = useUserPreferences({
    enabled: auth.isLoggedIn,
  });
  const user = useMemo(() => {
    if (!userQuery.data || !orgsQuery.data) {
      return;
    }

    const anonymousInfo: UserInfo = {
      userId: 0,
      alertingTourCompleted: false,
      productTourCompleted: false,
      snoozedAt: null,
      snoozeCount: 0,
      snoozedPmmVersion: '',
    };
    const anonymousPreferences: GetPreferenceResponse = {};

    const info = auth.isLoggedIn ? userInfoQuery.data : anonymousInfo;
    const preferences = auth.isLoggedIn
      ? preferencesQuery.data
      : anonymousPreferences;

    if (!info || !preferences) {
      return;
    }

    return getPerconaUser(
      userQuery.data,
      orgsQuery.data,
      info,
      preferences,
      isAuthorized(userQuery.error)
    );
  }, [auth.isLoggedIn, userQuery, orgsQuery, userInfoQuery, preferencesQuery]);

  return (
    <UserContext.Provider
      value={{
        isLoading:
          userQuery.isLoading || orgsQuery.isLoading || userInfoQuery.isLoading,
        user,
      }}
    >
      {children}
    </UserContext.Provider>
  );
};
