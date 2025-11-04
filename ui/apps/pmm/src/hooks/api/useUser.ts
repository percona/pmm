import {
  useMutation,
  UseMutationOptions,
  useQuery,
  useQueryClient,
  UseQueryOptions,
} from '@tanstack/react-query';
import {
  getCurrentUser,
  getCurrentUserOrgs,
  getUserInfo,
  updatePreferences,
  updateUserInfo,
} from 'api/user';
import { ApiError } from 'types/api.types';
import {
  GetUserResponse,
  UpdatePreferencesBody,
  UpdateUserInfoPayload,
  UserInfo,
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

export const useUserInfo = (options?: Partial<UseQueryOptions<UserInfo>>) =>
  useQuery({
    queryKey: ['user:me'],
    queryFn: () => getUserInfo(),
    ...options,
  });

export const useUpdateUserInfo = (
  options?: Partial<
    UseMutationOptions<UserInfo, ApiError, UpdateUserInfoPayload>
  >
) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationKey: ['user:me:update'],
    mutationFn: (payload) => updateUserInfo(payload),
    onSuccess: (data) => queryClient.setQueryData(['user:me'], data),
    ...options,
  });
};

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
