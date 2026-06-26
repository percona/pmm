import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  listInvestigations,
  getInvestigation,
  createInvestigation,
  patchInvestigation,
  deleteInvestigation,
  getInvestigationComments,
  getInvestigationMessages,
  getInvestigationTimeline,
  postInvestigationComment,
  postInvestigationChat,
  postInvestigationRun,
  patchInvestigationBlock,
  deleteInvestigationBlock,
  createServiceNowTicket,
  type CreateInvestigationBody,
  type PatchInvestigationBody,
  type CreateCommentBody,
  type PatchBlockBody,
} from 'api/investigations';

export const INVESTIGATIONS_KEYS = {
  all: ['investigations'] as const,
  list: (params?: { status?: string; trigger?: 'auto' | 'manual'; limit?: number; offset?: number; orderBy?: string; order?: 'asc' | 'desc' }) =>
    ['investigations', 'list', params] as const,
  detail: (id: string) => ['investigations', id] as const,
  comments: (id: string, blockId?: string) =>
    ['investigations', id, 'comments', blockId] as const,
  messages: (id: string, params?: { limit?: number; offset?: number }) =>
    ['investigations', id, 'messages', params] as const,
  /** Prefix for all message queries (matches any limit/offset params). */
  messagesPrefix: (id: string) => ['investigations', id, 'messages'] as const,
  timeline: (id: string) => ['investigations', id, 'timeline'] as const,
  usage: (id: string) => ['investigationUsage', id] as const,
};

export const useInvestigationsList = (params?: {
  status?: string;
  trigger?: 'auto' | 'manual';
  limit?: number;
  offset?: number;
  orderBy?: string;
  order?: 'asc' | 'desc';
  enabled?: boolean;
}) =>
  useQuery({
    queryKey: INVESTIGATIONS_KEYS.list(params),
    queryFn: () =>
      listInvestigations({
        status: params?.status,
        trigger: params?.trigger,
        limit: params?.limit,
        offset: params?.offset,
        orderBy: params?.orderBy,
        order: params?.order,
      }),
    enabled: params?.enabled ?? true,
  });

export const useInvestigation = (id: string | undefined, options?: { enabled?: boolean; refetchInterval?: number | false }) =>
  useQuery({
    queryKey: INVESTIGATIONS_KEYS.detail(id ?? ''),
    queryFn: () => getInvestigation(id!),
    enabled: (options?.enabled ?? true) && !!id,
    refetchInterval: options?.refetchInterval,
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

export const useDeleteInvestigation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => deleteInvestigation(id),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: INVESTIGATIONS_KEYS.all }),
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
  options?: { enabled?: boolean; refetchInterval?: number | false }
) =>
  useQuery({
    queryKey: INVESTIGATIONS_KEYS.messages(id ?? '', params),
    queryFn: () => getInvestigationMessages(id!, params),
    enabled: (options?.enabled ?? true) && !!id,
    refetchInterval: options?.refetchInterval,
  });

export const useInvestigationTimeline = (
  id: string | undefined,
  options?: { enabled?: boolean }
) =>
  useQuery({
    queryKey: INVESTIGATIONS_KEYS.timeline(id ?? ''),
    queryFn: () => getInvestigationTimeline(id!),
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
        queryKey: INVESTIGATIONS_KEYS.messagesPrefix(investigationId),
      });
      queryClient.invalidateQueries({
        queryKey: INVESTIGATIONS_KEYS.usage(investigationId),
      });
    },
  });
};

export const usePostInvestigationRun = (investigationId: string) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => postInvestigationRun(investigationId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: INVESTIGATIONS_KEYS.detail(investigationId),
      });
      queryClient.invalidateQueries({
        queryKey: INVESTIGATIONS_KEYS.messagesPrefix(investigationId),
      });
      queryClient.invalidateQueries({
        queryKey: INVESTIGATIONS_KEYS.timeline(investigationId),
      });
      queryClient.invalidateQueries({
        queryKey: INVESTIGATIONS_KEYS.usage(investigationId),
      });
    },
  });
};

export const usePatchInvestigationBlock = (investigationId: string) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      blockId,
      body,
    }: {
      blockId: string;
      body: PatchBlockBody;
    }) => patchInvestigationBlock(investigationId, blockId, body),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: INVESTIGATIONS_KEYS.detail(investigationId),
      });
    },
  });
};

export const useDeleteInvestigationBlock = (investigationId: string) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (blockId: string) =>
      deleteInvestigationBlock(investigationId, blockId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: INVESTIGATIONS_KEYS.detail(investigationId),
      });
    },
  });
};

export const useCreateServiceNowTicket = (investigationId: string) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => createServiceNowTicket(investigationId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: INVESTIGATIONS_KEYS.detail(investigationId),
      });
    },
  });
};
