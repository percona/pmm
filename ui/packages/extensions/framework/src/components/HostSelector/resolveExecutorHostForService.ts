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

import type { HostOption } from '../../hooks/useHosts';

/** Minimal service shape needed to resolve an executor host. */
export interface ServiceHostResolveInput {
  name: string;
  node?: { name: string; address: string } | null;
}

/**
 * Resolve an executor host for an inventory service.
 *
 * Mirrors the backend ``resolve_executor_host_for_service`` order:
 * 1. ``service.node.name`` against host ids (Nomad node names)
 * 2. ``service.node.address`` against host addresses
 * 3. ``service.name`` against host ids
 * 4. ``undefined`` when nothing matches (caller leaves the field empty)
 */
export function resolveExecutorHostForService(
  hosts: readonly HostOption[],
  service: ServiceHostResolveInput
): HostOption | undefined {
  const byId = new Map(hosts.map((host) => [host.id, host]));

  if (service.node?.name) {
    const byName = byId.get(service.node.name);
    if (byName) {
      return byName;
    }
  }

  if (service.node?.address) {
    const byAddress = hosts.find(
      (host) => host.address === service.node!.address
    );
    if (byAddress) {
      return byAddress;
    }
  }

  if (service.name) {
    return byId.get(service.name);
  }

  return undefined;
}
