import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  getAdreSettings,
  updateAdreSettings,
  getAdreModels,
  getAdreAlerts,
  getAdreDeployment,
  updateAdreDeploymentConfig,
  updateAdreDeploymentModels,
  deleteAdreDeploymentModel,
  updateAdreDeploymentPmmUrl,
  upsertAdreDeploymentSkill,
  deleteAdreDeploymentSkill,
  applyAdreDeployment,
  provisionAdreDeployment,
  type AdreSettings,
  type AdreDeploymentModelInput,
  type AdreDeploymentSkillInput,
} from 'api/adre';

export const ADRE_KEYS = {
  settings: ['adre', 'settings'] as const,
  models: ['adre', 'models'] as const,
  alerts: ['adre', 'alerts'] as const,
  deployment: ['adre', 'deployment'] as const,
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

// --- ADRE deployment config (admin-only) ---

export const useAdreDeployment = (options?: { enabled?: boolean }) =>
  useQuery({
    queryKey: ADRE_KEYS.deployment,
    queryFn: getAdreDeployment,
    enabled: options?.enabled ?? true,
  });

const useDeploymentMutation = <TArgs>(fn: (args: TArgs) => Promise<unknown>) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ADRE_KEYS.deployment }),
  });
};

export const useUpdateAdreDeploymentConfig = () =>
  useDeploymentMutation((configYaml: string) => updateAdreDeploymentConfig(configYaml));

export const useUpdateAdreDeploymentModels = () =>
  useDeploymentMutation((models: AdreDeploymentModelInput[]) => updateAdreDeploymentModels(models));

export const useDeleteAdreDeploymentModel = () =>
  useDeploymentMutation((name: string) => deleteAdreDeploymentModel(name));

export const useUpdateAdreDeploymentPmmUrl = () =>
  useDeploymentMutation((pmmUrl: string) => updateAdreDeploymentPmmUrl(pmmUrl));

export const useUpsertAdreDeploymentSkill = () =>
  useDeploymentMutation((skill: AdreDeploymentSkillInput) => upsertAdreDeploymentSkill(skill));

export const useDeleteAdreDeploymentSkill = () =>
  useDeploymentMutation((name: string) => deleteAdreDeploymentSkill(name));

export const useApplyAdreDeployment = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => applyAdreDeployment(),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ADRE_KEYS.deployment }),
  });
};

export const useProvisionAdreDeployment = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => provisionAdreDeployment(),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ADRE_KEYS.deployment }),
  });
};
