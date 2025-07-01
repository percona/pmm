import { useQuery, UseQueryOptions } from '@tanstack/react-query';
import { getCurrentUser } from 'api/user';
import { GetUserResponse } from 'types/user.types';

export const useCurrentUser = (
  options?: Partial<UseQueryOptions<GetUserResponse>>
) =>
  useQuery({
    queryKey: ['user'],
    queryFn: () => getCurrentUser(),
    ...options,
  });
