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
import Alert from '@mui/material/Alert';
import Card from '@mui/material/Card';
import CircularProgress from '@mui/material/CircularProgress';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Page } from 'components/page';
import { useUser } from 'contexts/user';
import { useReadonlySettings } from 'hooks/api/useSettings';
import { useLocalStorage } from 'hooks/utils/useLocalStorage';
import { OrgRole } from 'types/user.types';

const TECHNICAL_PREVIEW_DISMISSED_KEY = 'pmm-ui.om.technicalPreviewDismissed';

const Messages = {
  switchedOff:
    'OpenManager is switched off. A PMM admin can turn it on in Configuration → Settings → Advanced Settings.',
  technicalPreview: 'Technical preview',
  technicalPreviewBody:
    'OpenManager is a technical preview. It is still under development and may change.',
};

/**
 * Host chrome for the OM page.
 *
 * Deliberately not `SepPage`, which OM used while its backend was a SEP app. That
 * wrapper holds its children behind `SepAuthGate` until a SEP bearer has been minted
 * from the PMM session, and fails closed — so a SEP that is down, unconfigured or
 * refusing the exchange would blank a page whose data comes from pmm-managed's own
 * inventory, VictoriaMetrics and stored snapshot. OM has no SEP call on any browser
 * path, so it must not inherit SEP's availability.
 *
 * The PMM-admin restriction is kept, and enforced here rather than left to the
 * sidebar: NavigationProvider only *hides* the entry for non-admins, while the route
 * still matches on direct navigation. The predicate is the nav's own — `isPMMAdmin` is
 * `isGrafanaAdmin || orgRole === Admin`, and `roles` (org-role only) cannot express the
 * Grafana-admin half on its own, so it gates the remaining case and `Page` renders its
 * standard unauthorized card.
 *
 * The settings gate is enforced here for the same reason: a saved or shared link to
 * this page has to answer "switched off", not the API's raw FailedPrecondition, and
 * not the unauthorized card above, which would misreport a disabled feature as a
 * permissions problem to an admin who has every right to be here (PMM-15360 AC1/AC2/AC7).
 *
 * The technical-preview banner is dismissible, and remembered per browser rather than
 * per PMM account or installation -- it is a "you've seen this" acknowledgement, not a
 * setting with a right answer for every viewer, so localStorage is enough and needs no
 * round trip to pmm-managed.
 *
 * The close button's `sx` override exists because `@percona/peak-ui`'s MuiAlert theme
 * (`styleOverrides.icon`/`.message`) sets `color: theme.palette[severity].contrastText`
 * on the icon and message slots, but not on `.MuiAlert-action` -- so the close button
 * MUI renders for `onClose` falls back to the alert root's own `color`, which this
 * theme leaves close to the warning background itself. Nothing else in this app uses a
 * dismissible Alert, which is presumably why that gap was never hit before.
 */
export const OmPage: FC<PropsWithChildren> = ({ children }) => {
  const { user } = useUser();
  const { data: settings, isLoading } = useReadonlySettings();
  const [previewDismissed, setPreviewDismissed] = useLocalStorage<boolean>(
    TECHNICAL_PREVIEW_DISMISSED_KEY,
    false
  );

  return (
    <Page
      maxWidth="full"
      roles={user?.isPMMAdmin ? undefined : [OrgRole.Admin]}
    >
      <Stack gap={3} sx={{ flex: 1 }}>
        {isLoading ? (
          <Stack alignItems="center" py={4}>
            <CircularProgress data-testid="om-loading" />
          </Stack>
        ) : settings?.omEnabled ? (
          <>
            {!previewDismissed && (
              <Alert
                severity="warning"
                onClose={() => setPreviewDismissed(true)}
                data-testid="om-technical-preview"
                sx={{
                  '& .MuiAlert-action': {
                    color: (theme) => theme.palette.warning.contrastText,
                  },
                }}
              >
                <Typography variant="body2">
                  <strong>{Messages.technicalPreview}</strong>{' '}
                  {Messages.technicalPreviewBody}
                </Typography>
              </Alert>
            )}
            <div>{children}</div>
          </>
        ) : (
          <Card variant="outlined" sx={{ p: 2 }}>
            <Alert severity="info" data-testid="om-switched-off">
              {Messages.switchedOff}
            </Alert>
          </Card>
        )}
      </Stack>
    </Page>
  );
};
