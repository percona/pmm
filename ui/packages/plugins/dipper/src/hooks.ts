/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient, type PluginSchema } from '@sep/api';
import type { PaginatedTaskHistory } from '@sep/framework';
import type { DipperCollectorType, DipperExecutionResponse, DipperExecutionWrite } from './types';

const DIPPER_BASE = '/apps/dipper';

export function useDipperAppSchema() {
  return useQuery<PluginSchema>({
    queryKey: ['dipper', 'schema'],
    queryFn: async () => {
      const { data } = await apiClient.get<PluginSchema>(`${DIPPER_BASE}/schema`);
      return data;
    },
    staleTime: 5 * 60 * 1000,
  });
}

export function useDipperFormSchema(serviceId: number | null, collectorType: DipperCollectorType) {
  return useQuery<PluginSchema>({
    queryKey: ['dipper', 'form-schema', serviceId, collectorType],
    enabled: serviceId !== null,
    queryFn: async () => {
      const { data } = await apiClient.get<PluginSchema>(`${DIPPER_BASE}/form-schema`, {
        params: { service_id: serviceId, collector_type: collectorType },
      });
      return data;
    },
  });
}

export function useDipperHistory() {
  return useQuery<PaginatedTaskHistory>({
    queryKey: ['dipper', 'history'],
    queryFn: async () => {
      const { data } = await apiClient.get<PaginatedTaskHistory>(`${DIPPER_BASE}/`);
      return data;
    },
  });
}

export function useDipperExecution() {
  const queryClient = useQueryClient();
  return useMutation<DipperExecutionResponse, Error, DipperExecutionWrite>({
    mutationFn: async (body) => {
      const { data } = await apiClient.post<DipperExecutionResponse>(`${DIPPER_BASE}/`, body);
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dipper', 'history'] });
    },
  });
}
