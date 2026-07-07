import { FC, PropsWithChildren } from 'react';
import Stack from '@mui/material/Stack';
import { Page } from 'components/page';

/**
 * Shared container for SEP apps mounted as native PMM routes.
 *
 * Wraps a plugin (which composes its own <Routes> and renders its own
 * heading) in the same Page + Stack chrome PMM pages use (see
 * pages/settings/Settings.tsx), so SEP content gets the standard PMM
 * padding, width, auth gate, and footer. No `title` is passed: the SEP
 * plugins already render their own headings.
 */
export const SepPage: FC<PropsWithChildren> = ({ children }) => (
  <Page fullWidth>
    <Stack gap={3} sx={{ flex: 1 }}>
      <div>{children}</div>
    </Stack>
  </Page>
);
