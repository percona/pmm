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

import { FC, PropsWithChildren } from 'react';
import Stack from '@mui/material/Stack';
import { Page } from 'components/page';
import { useUser } from 'contexts/user';
import { OrgRole } from 'types/user.types';

/**
 * Host chrome for the POM page.
 *
 * Deliberately not `SepPage`, which POM used while its backend was a SEP app. That
 * wrapper holds its children behind `SepAuthGate` until a SEP bearer has been minted
 * from the PMM session, and fails closed — so a SEP that is down, unconfigured or
 * refusing the exchange would blank a page whose data comes from pmm-managed's own
 * inventory, VictoriaMetrics and stored snapshot. POM has no SEP call on any browser
 * path, so it must not inherit SEP's availability.
 *
 * The PMM-admin restriction is kept, and enforced here rather than left to the
 * sidebar: NavigationProvider only *hides* the entry for non-admins, while the route
 * still matches on direct navigation. The predicate is the nav's own — `isPMMAdmin` is
 * `isGrafanaAdmin || orgRole === Admin`, and `roles` (org-role only) cannot express the
 * Grafana-admin half on its own, so it gates the remaining case and `Page` renders its
 * standard unauthorized card.
 */
export const PomPage: FC<PropsWithChildren> = ({ children }) => {
  const { user } = useUser();

  return (
    <Page
      maxWidth="full"
      roles={user?.isPMMAdmin ? undefined : [OrgRole.Admin]}
    >
      <Stack gap={3} sx={{ flex: 1 }}>
        <div>{children}</div>
      </Stack>
    </Page>
  );
};
