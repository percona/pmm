import Box from '@mui/material/Box';
import { PageHeader } from '@percona/peak-ui';
import type { FC } from 'react';
import { ShareLinkButton } from './ShareLinkButton';
import { StarAppButton } from './StarAppButton';

export interface AppHeaderProps {
  title: string;
  /** The app's id when it can be starred: only the apps under Apps can, not those with a place of their own. */
  starId?: string;
  /** `page` lines it up with a full-width PMM page under it; `frame` with a framed Grafana page or the Real-time pages, 16px in. */
  inset: 'page' | 'frame';
}

/** The top of every app, Query analytics and Real-time queries included: its name, a star for the apps under Apps, and a link to share. */
export const AppHeader: FC<AppHeaderProps> = ({ title, starId, inset }) => (
  <Box
    sx={{
      pt: 2,
      // PageContainer's full-width padding, so the title lines up with the page under it.
      px: inset === 'page' ? { xs: 2, md: 4 } : 2,
    }}
    data-testid="app-header"
  >
    <PageHeader
      title={title}
      divider
      actions={
        <Box sx={{ display: 'flex' }}>
          {starId && <StarAppButton id={starId} />}
          <ShareLinkButton />
        </Box>
      }
    />
  </Box>
);
