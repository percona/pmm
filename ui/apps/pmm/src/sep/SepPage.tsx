import { FC, PropsWithChildren } from 'react';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import { Page } from 'components/page';
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
 */
export const SepPage: FC<PropsWithChildren> = ({ children }) => (
  <Page maxWidth="full">
    <Stack gap={3} sx={{ flex: 1 }}>
      <SepAuthProvider>
        <SepAuthGate>
          {/*
            A flex column that grows, not a plain block: it carries the height
            handed down from Page so a plugin (or the ServiceNow setup prompt)
            can centre itself in the page rather than in its own content box.
          */}
          <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
            {children}
          </Box>
        </SepAuthGate>
      </SepAuthProvider>
    </Stack>
  </Page>
);
