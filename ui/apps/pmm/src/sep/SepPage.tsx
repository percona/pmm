import { FC, PropsWithChildren } from 'react';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import CircularProgress from '@mui/material/CircularProgress';
import Stack from '@mui/material/Stack';
import { Page } from 'components/page';
import { useSettings } from 'contexts/settings';
import { SepAuthGate } from './SepAuthGate';
import { SepAuthProvider } from './SepAuthProvider';

/**
 * Shared container for SEP apps mounted as native PMM routes.
 *
 * Wraps a plugin (which composes its own <Routes> and renders its own
 * heading) in the same Page + Stack chrome PMM pages use (see
 * pages/settings/Settings.tsx), so SEP content gets the standard PMM
 * padding, width, auth gate, and footer. No `title` is passed: the SEP
 * plugins already render their own headings.
 *
 * No `roles` restriction: every signed-in PMM user may open a SEP page, and
 * what they can do there is decided per control rather than per route. SEP's
 * API admits any authenticated session to its reads and holds every unsafe
 * method to administrators, so a non-admin gets the lists, details, logs and
 * history with no write control offered (PMM-15358). The admin-only route
 * guard this replaced predated that per-control gating and closed the
 * read-only view the API was always willing to serve.
 *
 * `SepAuthProvider` wraps the whole subtree so framework and plugin components
 * can read the session's mutation capability. It sits outside `SepAuthGate`: a
 * component rendered while the exchange is still in flight must resolve the
 * same capability it will hold once the bearer lands, not the non-admin
 * fallback.
 *
 * When SEP is disabled (`sepEnabled` from server settings), direct navigation
 * to a SEP route renders an explicit unavailable state instead of mounting the
 * auth gate or firing SEP API requests. Settings must resolve first — otherwise
 * a hard refresh flashes "not enabled" while `settings` is still null.
 */
export const SepPage: FC<PropsWithChildren> = ({ children }) => {
  const { settings, isLoading } = useSettings();

  if (isLoading || !settings) {
    return (
      <Page maxWidth="full">
        <Stack alignItems="center" py={4}>
          <CircularProgress data-testid="sep-settings-loading" />
        </Stack>
      </Page>
    );
  }

  if (!settings.sepEnabled) {
    return (
      <Page maxWidth="full">
        <Alert severity="info">
          This feature is not enabled. Contact your administrator.
        </Alert>
      </Page>
    );
  }

  return (
    <Page maxWidth="full">
      <Stack gap={3} sx={{ flex: 1 }}>
        <SepAuthProvider>
          <SepAuthGate>
            {/*
              A flex column that grows, not a plain block: it carries the height
              handed down from Page so a plugin can centre itself in the page
              rather than in its own content box.
            */}
            <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
              {children}
            </Box>
          </SepAuthGate>
        </SepAuthProvider>
      </Stack>
    </Page>
  );
};
