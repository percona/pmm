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

import { useHosts } from '../../hooks/useHosts';
import { useSchemas } from '../../hooks/useSchemas';
import { useServices, type ServiceType } from '../../hooks/useServices';
import { useTables } from '../../hooks/useTables';
import { extractId } from '../../utils/extractId';
import { getServiceOptionLabel } from '../ServiceSelector/ServiceSelector';
import type { SettingReference } from './taskConfiguration';

const LOADING = 'Loading…';

interface InventoryLookup<T> {
  data: readonly T[] | undefined;
  isLoading: boolean;
}

function labelById<T extends { id: number }>(
  lookup: InventoryLookup<T>,
  id: number | null,
  value: unknown,
  label: (option: T) => string
): string {
  if (id === null) {
    return String(value);
  }
  if (lookup.isLoading) {
    return LOADING;
  }
  const match = lookup.data?.find((option) => option.id === id);
  return match ? label(match) : `Unknown (inventory ID ${id})`;
}

const byName = (option: { name: string }) => option.name;

/**
 * The name a stored inventory reference stands for, as the form's selector
 * showed it: a service as `name (type)`, a host, schema or table by name.
 *
 * A stored value that is not an id was typed in by hand on an `allow_custom`
 * field, and reads as typed. An id the lookup cannot place — gone from
 * inventory since, or the lookup failed — says so instead of printing the bare
 * id. A host that cannot be placed keeps its stored id, which is the executor's
 * node name and already reads as one.
 */
export function useReferenceLabel(
  reference: SettingReference,
  value: unknown
): string {
  const id = reference.kind === 'host' ? null : extractId(value);

  const services = useServices({
    serviceTypes:
      reference.kind === 'service'
        ? (reference.serviceTypes as readonly ServiceType[])
        : undefined,
    enabled: reference.kind === 'service' && id !== null,
  });
  const hosts = useHosts({ enabled: reference.kind === 'host' });
  const schemas = useSchemas({
    serviceId: reference.kind === 'schema' ? reference.serviceId : null,
    enabled: id !== null,
  });
  const tables = useTables({
    schemaId: reference.kind === 'table' ? reference.schemaId : null,
    enabled: id !== null,
  });

  switch (reference.kind) {
    case 'service':
      return labelById(services, id, value, getServiceOptionLabel);
    case 'schema':
      return labelById(schemas, id, value, byName);
    case 'table':
      return labelById(tables, id, value, byName);
    case 'host': {
      const hostId = String(value);
      if (hosts.isLoading) {
        return LOADING;
      }
      return hosts.data?.find((host) => host.id === hostId)?.name ?? hostId;
    }
  }
}
