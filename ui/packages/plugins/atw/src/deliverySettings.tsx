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

import { FC, PropsWithChildren, createContext, useContext } from 'react';

/**
 * Where the embedding shell keeps the ServiceNow delivery settings.
 *
 * The app can tell that sending is unavailable — the backend says so in
 * `send_disabled_reasons` — but not where the shell parks the form that fixes
 * it: PMM owns a dedicated settings tab, SEP has only its generic settings
 * list. So the route arrives from the host rather than being assumed here, and
 * a host that passes nothing simply gets the explanation without the link.
 *
 * Router-relative and rendered as a `<Link>`, so it must resolve inside the
 * same router the app is mounted in.
 */
const DeliverySettingsContext = createContext<string | undefined>(undefined);

export const DeliverySettingsProvider: FC<
  PropsWithChildren<{ path?: string }>
> = ({ path, children }) => (
  <DeliverySettingsContext.Provider value={path}>
    {children}
  </DeliverySettingsContext.Provider>
);

/** The host's delivery settings route, or `undefined` when it offers none. */
export function useDeliverySettingsPath(): string | undefined {
  return useContext(DeliverySettingsContext);
}
