import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  getAdreSettings,
  updateAdreSettings,
  getAdreModels,
  getAdreAlerts,
  type AdreSettings,
} from 'api/adre';

export const ADRE_KEYS = {
  settings: ['adre', 'settings'] as const,
  models: ['adre', 'models'] as const,
  alerts: ['adre', 'alerts'] as const,
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

export const useAdreAlerts = (options?: { enabled?: boolean }) => {
  const query = useQuery({
    queryKey: ADRE_KEYS.alerts,
    queryFn: async () => {
      const data = (await getAdreAlerts()) as { data?: { alerts?: unknown[] }; alerts?: unknown[] };
      const list = data?.data?.alerts ?? data?.alerts ?? [];
      return Array.isArray(list) ? list : [];
    },
    enabled: options?.enabled ?? true,
  });
  return { ...query, alerts: query.data ?? [] };
};
