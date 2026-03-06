import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  getAdreSettings,
  updateAdreSettings,
  getAdreModels,
  type AdreSettings,
  type AdreChatRequest,
  type AdreInvestigateRequest,
} from 'api/adre';

export const ADRE_KEYS = {
  settings: ['adre', 'settings'] as const,
  models: ['adre', 'models'] as const,
};

export const useAdreSettings = (options?: { enabled?: boolean }) =>
  useQuery({
    queryKey: ADRE_KEYS.settings,
    queryFn: getAdreSettings,
    ...options,
  });

export const useUpdateAdreSettings = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: Partial<AdreSettings>) => updateAdreSettings(body),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ADRE_KEYS.settings }),
  });
};

export const useAdreModels = (options?: { enabled?: boolean }) =>
  useQuery({
    queryKey: ADRE_KEYS.models,
    queryFn: getAdreModels,
    enabled: (options?.enabled ?? true),
  });
