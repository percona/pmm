import { FC, PropsWithChildren, useMemo } from 'react';
import { UserContext } from './user.context';
import { useCurrentUser } from 'hooks/api/useUser';
import { useFrontendSettings, useSettings } from 'hooks/api/useSettings';
import { getPerconaUser, isAuthorized } from './user.utils';

export const UserProvider: FC<PropsWithChildren> = ({ children }) => {
  const frontendSettings = useFrontendSettings();
  const { data, isLoading: isLoadingUser } = useCurrentUser({
    enabled: !frontendSettings.data?.anonymousEnabled,
  });
  const { error, isLoading: isLoadingSettings } = useSettings({
    retry: false,
    enabled: !frontendSettings.data?.anonymousEnabled,
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
