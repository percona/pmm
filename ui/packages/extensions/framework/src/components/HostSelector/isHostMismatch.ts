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

/**
 * Decide whether the selected executor host and the target service's node
 * live at different network addresses.
 *
 * Co-location is address-only: Nomad node names and inventory display names
 * are independent namespaces, so two hosts sharing one address are the same
 * machine and must not warn. An absent or empty address is unknown, not a
 * difference — prefer silence whenever co-location cannot be established.
 */
export function isHostMismatch(
  hostAddress: string | undefined,
  serviceNodeAddress: string | undefined
): boolean {
  if (!hostAddress || !serviceNodeAddress) {
    return false;
  }
  return hostAddress !== serviceNodeAddress;
}
