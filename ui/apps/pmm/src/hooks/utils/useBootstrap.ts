import { useAuth } from 'contexts/auth';
import { useSettings } from 'contexts/settings';
import { useUser } from 'contexts/user';

export const useBootstrap = () => {
  const { isLoading: isLoadingAuth } = useAuth();
  const { isLoading: isLoadingUser } = useUser();
  const { isLoading: isSettingsLoading } = useSettings();

  return { isReady: !(isLoadingAuth || isLoadingUser || isSettingsLoading) };
};
