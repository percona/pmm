import { FC, PropsWithChildren } from 'react';
import Stack from '@mui/material/Stack';
import { Page } from 'components/page';
import { useUser } from 'contexts/user';
import { OrgRole } from 'types/user.types';
import { SepAuthGate } from './SepAuthGate';

/**
 * Shared container for SEP apps mounted as native PMM routes.
 *
 * Wraps a plugin (which composes its own <Routes> and renders its own
 * heading) in the same Page + Stack chrome PMM pages use (see
 * pages/settings/Settings.tsx), so SEP content gets the standard PMM
 * padding, width, auth gate, and footer. No `title` is passed: the SEP
 * plugins already render their own headings.
 *
 * The PMM-admin restriction is enforced here rather than left to the sidebar:
 * NavigationProvider only *hides* the SEP entries for non-admins, while the
 * routes still match on direct navigation. It reuses the nav's own predicate:
 * `isPMMAdmin` is `isGrafanaAdmin || orgRole === Admin`, and `roles` (org-role
 * only) cannot express the Grafana-admin half on its own, so it gates the
 * remaining case and Page renders its standard unauthorized card.
 *
 * `SepAuthGate` sits inside that check, so the SEP session exchange only runs
 * for a user who is allowed on the page in the first place.
 */
export const SepPage: FC<PropsWithChildren> = ({ children }) => {
  const { user } = useUser();

  return (
    <Page
      maxWidth="full"
      roles={user?.isPMMAdmin ? undefined : [OrgRole.Admin]}
    >
      <Stack gap={3} sx={{ flex: 1 }}>
        <SepAuthGate>
          <div>{children}</div>
        </SepAuthGate>
      </Stack>
    </Page>
  );
};
