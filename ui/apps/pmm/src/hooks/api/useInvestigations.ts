import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  listInvestigations,
  getInvestigation,
  createInvestigation,
  patchInvestigation,
  getInvestigationComments,
  getInvestigationMessages,
  postInvestigationComment,
  postInvestigationChat,
  type CreateInvestigationBody,
  type PatchInvestigationBody,
  type CreateCommentBody,
} from 'api/investigations';

export const INVESTIGATIONS_KEYS = {
  all: ['investigations'] as const,
  list: (params?: { status?: string }) =>
    ['investigations', 'list', params] as const,
  detail: (id: string) => ['investigations', id] as const,
  comments: (id: string, blockId?: string) =>
    ['investigations', id, 'comments', blockId] as const,
  messages: (id: string, params?: { limit?: number; offset?: number }) =>
    ['investigations', id, 'messages', params] as const,
};

export const useInvestigationsList = (params?: {
  status?: string;
  limit?: number;
  offset?: number;
  enabled?: boolean;
}) =>
  useQuery({
    queryKey: INVESTIGATIONS_KEYS.list(params),
    queryFn: () =>
      listInvestigations({
        status: params?.status,
        limit: params?.limit,
        offset: params?.offset,
      }),
    enabled: params?.enabled ?? true,
  });

export const useInvestigation = (id: string | undefined, options?: { enabled?: boolean }) =>
  useQuery({
    queryKey: INVESTIGATIONS_KEYS.detail(id ?? ''),
    queryFn: () => getInvestigation(id!),
    enabled: (options?.enabled ?? true) && !!id,
  });

export const useCreateInvestigation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: CreateInvestigationBody) => createInvestigation(body),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: INVESTIGATIONS_KEYS.all }),
  });
};

export const usePatchInvestigation = (id: string) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: PatchInvestigationBody) => patchInvestigation(id, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: INVESTIGATIONS_KEYS.detail(id) });
      queryClient.invalidateQueries({ queryKey: INVESTIGATIONS_KEYS.all });
    },
  });
};

export const useInvestigationComments = (
  id: string | undefined,
  blockId?: string,
  options?: { enabled?: boolean }
) =>
  useQuery({
    queryKey: INVESTIGATIONS_KEYS.comments(id ?? '', blockId),
    queryFn: () => getInvestigationComments(id!, blockId),
    enabled: (options?.enabled ?? true) && !!id,
  });

export const useInvestigationMessages = (
  id: string | undefined,
  params?: { limit?: number; offset?: number },
  options?: { enabled?: boolean }
) =>
  useQuery({
    queryKey: INVESTIGATIONS_KEYS.messages(id ?? '', params),
    queryFn: () => getInvestigationMessages(id!, params),
    enabled: (options?.enabled ?? true) && !!id,
  });

export const usePostInvestigationComment = (investigationId: string) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: CreateCommentBody) =>
      postInvestigationComment(investigationId, body),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: INVESTIGATIONS_KEYS.comments(investigationId),
      });
    },
  });
};

export const usePostInvestigationChat = (investigationId: string) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (message: string) =>
      postInvestigationChat(investigationId, { message }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: INVESTIGATIONS_KEYS.detail(investigationId),
      });
      queryClient.invalidateQueries({
        queryKey: INVESTIGATIONS_KEYS.messages(investigationId),
      });
    },
  });
};
