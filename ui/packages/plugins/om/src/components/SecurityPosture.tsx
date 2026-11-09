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

import { Alert, AlertTitle } from '@mui/material';

/**
 * The security posture of a developer-preview install, stated where it is read.
 *
 * This used to be a "Security" tab holding three permanently disabled text fields -
 * "inputs that are not inputs", which was the complaint. A tab that can never be
 * edited is worse than the same facts stated where the user is already looking, so the
 * tab is gone and this renders on the Configure step, again at Review, and once more
 * on a finished install's completion card - one wording in all three, so what the
 * user agreed to is what they are told they got.
 *
 * It must keep naming the mechanism and what is unavailable: the tab was the only place
 * that said keyFile, LDAP and KMIP/KMS at all, and dropping that would make the
 * posture read as better than it is.
 */
export const SecurityPosture = ({
  title = 'Security in this developer preview',
}: {
  title?: string;
}) => (
  <Alert severity="info">
    <AlertTitle>{title}</AlertTitle>
    Members authenticate to each other with a shared keyFile, and client
    connections are <strong>not encrypted</strong> - TLS is off. LDAP, KMIP/KMS
    and encryption at rest are not available yet, and none of them can be
    configured here.
  </Alert>
);
