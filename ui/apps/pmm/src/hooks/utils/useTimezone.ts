import { useUser } from 'contexts/user/user.hooks';

export const useTimezone = () => {
  const { user } = useUser();
  return user?.preferences?.timezone || 'UTC';
};
