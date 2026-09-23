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

import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import { apiClient, type InventoryComponents } from '@sep/api';
import { sepRetry } from './sepRetry';

export type ServiceType = InventoryComponents['schemas']['ServiceTypeEnum'];

/** Inventory node fields needed to cascade an executor host from a service. */
export interface ServiceNodeOption {
  name: string;
  address: string;
}

export interface ServiceOption {
  id: number;
  name: string;
  type: ServiceType;
  /** Present when the list response includes the nested inventory node. */
  node?: ServiceNodeOption;
}

type PaginatedServices =
  InventoryComponents['schemas']['PaginatedResponse_ServiceResponse_'];

export interface UseServicesOptions {
  serviceTypes?: readonly ServiceType[];
  enabled?: boolean;
}

const DEFAULT_LIMIT = 200;
// Upper bound on paginated fetch iterations. At DEFAULT_LIMIT=200 this caps
// the response set at 10 000 services per type, well above any realistic
// customer scale. The guard fires only when the backend reports `items` and
// `total` in an inconsistent way that would otherwise spin forever.
const MAX_PAGES = 50;

const STABLE_EMPTY: readonly ServiceType[] = Object.freeze([] as ServiceType[]);

function normaliseTypes(
  types: readonly ServiceType[] | undefined
): readonly ServiceType[] {
  if (!types || types.length === 0) {
    return STABLE_EMPTY;
  }
  return [...types].sort();
}

async function fetchServicesPage(
  serviceType: ServiceType | undefined,
  offset: number
): Promise<PaginatedServices> {
  const params: Record<string, string | number> = {
    offset,
    limit: DEFAULT_LIMIT,
  };
  if (serviceType) {
    params.service_type = serviceType;
  }
  const { data } = await apiClient.get<PaginatedServices>('/sep/services/', {
    params,
  });
  return data;
}

async function fetchServicesForType(
  serviceType: ServiceType | undefined
): Promise<ServiceOption[]> {
  const out: ServiceOption[] = [];
  let offset = 0;
  for (let iter = 0; iter < MAX_PAGES; iter++) {
    const page = await fetchServicesPage(serviceType, offset);
    for (const svc of page.items) {
      if (svc.id !== null && svc.id !== undefined) {
        out.push({
          id: svc.id,
          name: svc.name,
          type: svc.type,
          ...(svc.node
            ? { node: { name: svc.node.name, address: svc.node.address } }
            : {}),
        });
      }
    }
    offset += page.items.length;
    if (offset >= page.total || page.items.length === 0) {
      return out;
    }
  }
  throw new Error(
    `useServices: pagination exceeded ${MAX_PAGES} pages for service_type=${serviceType ?? 'all'}; backend likely reports an inconsistent total/items pair`
  );
}

/**
 * Fetch services through the SEP gateway (`/api/sep/services/`), optionally
 * filtered to one or more service types. Multiple types fan out to parallel
 * paginated requests because the upstream `/services/` endpoint accepts a
 * single `service_type` only. The frontend must not call `/api/inventory/`
 * directly (see `app/sep/api/router.py`).
 */
export function useServices(
  options: UseServicesOptions = {}
): UseQueryResult<ServiceOption[], Error> {
  const { enabled = true } = options;
  const types = normaliseTypes(options.serviceTypes);
  return useQuery<ServiceOption[], Error>({
    queryKey: ['sep', 'services', { types }],
    enabled,
    staleTime: 60_000,
    retry: sepRetry,
    queryFn: async () => {
      if (types.length === 0) {
        return fetchServicesForType(undefined);
      }
      const pages = await Promise.all(
        types.map((t) => fetchServicesForType(t))
      );
      const seen = new Set<number>();
      const out: ServiceOption[] = [];
      for (const page of pages) {
        for (const svc of page) {
          if (!seen.has(svc.id)) {
            seen.add(svc.id);
            out.push(svc);
          }
        }
      }
      return out;
    },
  });
}
