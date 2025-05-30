import { useQuery } from '@tanstack/react-query';
import { getCurrentUser } from 'api/user';

export const useCurrentUser = () =>
  useQuery({
    queryKey: ['user'],
    queryFn: () => getCurrentUser(),
  });
