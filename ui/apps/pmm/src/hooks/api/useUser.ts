import {
  useMutation,
  UseMutationOptions,
  useQuery,
  UseQueryOptions,
} from '@tanstack/react-query';
import {
  getCurrentUser,
  getCurrentUserOrgs,
  updatePreferences,
} from 'api/user';
import { ApiError } from 'types/api.types';
import {
  GetUserResponse,
  UpdatePreferencesBody,
  UserOrg,
} from 'types/user.types';

export const useCurrentUser = (
  options?: Partial<UseQueryOptions<GetUserResponse>>
) =>
  useQuery({
    queryKey: ['user'],
    queryFn: () => getCurrentUser(),
    ...options,
  });

export const useCurrentUserOrgs = (
  options?: Partial<UseQueryOptions<UserOrg[]>>
) =>
  useQuery({
    queryKey: ['user:orgs'],
    queryFn: () => getCurrentUserOrgs(),
    ...options,
  });

export const useUpdatePreferences = (
  options?: Partial<UseMutationOptions<void, ApiError, UpdatePreferencesBody>>
) =>
  useMutation({
    mutationKey: ['user:preferences'],
    mutationFn: (preferences: Partial<UpdatePreferencesBody>) =>
      updatePreferences(preferences),
    ...options,
  });
