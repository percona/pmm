import { useAuth } from 'contexts/auth';
import { useUser } from 'contexts/user/user.hooks';

export const useBootstrap = () => {
  const { isLoading: isLoadingAuth } = useAuth();
  const { isLoading: isLoadingUser } = useUser();

  return { isReady: !isLoadingAuth && !isLoadingUser };
};
