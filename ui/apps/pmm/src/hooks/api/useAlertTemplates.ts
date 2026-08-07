import {
  useMutation,
  UseMutationOptions,
  useQuery,
  useQueryClient,
  UseQueryOptions,
} from '@tanstack/react-query';
import {
  createRuleFromTemplate,
  createTemplate,
  deleteTemplate,
  listTemplates,
  updateTemplate,
} from 'api/alert-templates';
import {
  CreateRulePayload,
  CreateTemplatePayload,
  DeleteTemplatePayload,
  ListTemplatesParams,
  ListTemplatesResponse,
  UpdateTemplatePayload,
} from 'types/alert-templates.types';
import { EmptyResponse } from 'types/util.types';

const KEYS = {
  LIST_TEMPLATES: 'alert-templates:list',
  CREATE_TEMPLATE: 'alert-templates:create',
  UPDATE_TEMPLATE: 'alert-templates:update',
  DELETE_TEMPLATE: 'alert-templates:delete',
  CREATE_RULE: 'alert-templates:create-rule',
};

export const useAlertTemplates = (
  params?: ListTemplatesParams,
  options?: Partial<UseQueryOptions<ListTemplatesResponse>>
) =>
  useQuery({
    queryKey: [KEYS.LIST_TEMPLATES, params],
    queryFn: () => listTemplates(params),
    ...options,
  });

export const useCreateTemplate = (
  options?: Partial<
    UseMutationOptions<EmptyResponse, Error, CreateTemplatePayload>
  >
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: [KEYS.CREATE_TEMPLATE],
    mutationFn: createTemplate,
    ...options,
    onSuccess: async (data, variables, onMutate, context) => {
      await options?.onSuccess?.(data, variables, onMutate, context);
      await queryClient.invalidateQueries({
        queryKey: [KEYS.LIST_TEMPLATES],
      });
    },
  });
};

export const useUpdateTemplate = (
  options?: Partial<
    UseMutationOptions<EmptyResponse, Error, UpdateTemplatePayload>
  >
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: [KEYS.UPDATE_TEMPLATE],
    mutationFn: updateTemplate,
    ...options,
    onSuccess: async (data, variables, onMutate, context) => {
      await options?.onSuccess?.(data, variables, onMutate, context);
      await queryClient.invalidateQueries({
        queryKey: [KEYS.LIST_TEMPLATES],
      });
    },
  });
};

export const useDeleteTemplate = (
  options?: Partial<
    UseMutationOptions<EmptyResponse, Error, DeleteTemplatePayload>
  >
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: [KEYS.DELETE_TEMPLATE],
    mutationFn: deleteTemplate,
    ...options,
    onSuccess: async (data, variables, onMutate, context) => {
      await options?.onSuccess?.(data, variables, onMutate, context);
      await queryClient.invalidateQueries({
        queryKey: [KEYS.LIST_TEMPLATES],
      });
    },
  });
};

export const useCreateRuleFromTemplate = (
  options?: Partial<UseMutationOptions<EmptyResponse, Error, CreateRulePayload>>
) =>
  useMutation({
    mutationKey: [KEYS.CREATE_RULE],
    mutationFn: createRuleFromTemplate,
    ...options,
  });
