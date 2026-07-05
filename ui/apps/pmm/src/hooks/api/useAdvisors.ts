import {
  useMutation,
  UseMutationOptions,
  useQuery,
  useQueryClient,
  UseQueryOptions,
} from '@tanstack/react-query';
import {
  changeAdvisorChecks,
  listAdvisors,
  startAdvisorChecks,
} from 'api/advisors';
import { Advisor, ChangeAdvisorCheckParams } from 'types/advisors.types';

const KEYS = {
  LIST: 'advisors:list',
  START_CHECKS: 'advisors:start-checks',
  CHANGE_CHECKS: 'advisors:change-checks',
};

export const useAdvisors = (options?: Partial<UseQueryOptions<Advisor[]>>) =>
  useQuery({
    queryKey: [KEYS.LIST],
    queryFn: () => listAdvisors(),
    ...options,
  });

export const useStartAdvisorChecks = (
  options?: Partial<UseMutationOptions<void, Error, string[]>>
) =>
  useMutation({
    mutationKey: [KEYS.START_CHECKS],
    mutationFn: startAdvisorChecks,
    ...options,
  });

export const useChangeAdvisorChecks = (
  options?: Partial<
    UseMutationOptions<void, Error, ChangeAdvisorCheckParams[]>
  >
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: [KEYS.CHANGE_CHECKS],
    mutationFn: changeAdvisorChecks,
    ...options,
    onSuccess: async (data, variables, onMutate, context) => {
      await options?.onSuccess?.(data, variables, onMutate, context);
      await queryClient.invalidateQueries({ queryKey: [KEYS.LIST] });
    },
  });
};
