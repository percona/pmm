import { FC } from 'react';
import { useLocation, useParams } from 'react-router-dom';
import Chip from '@mui/material/Chip';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Page } from 'components/page';
import { useNavigation } from 'contexts/navigation';
import { findAppPage } from 'contexts/navigation/navigation.proposal';
import { findActiveNavItem } from 'utils/navigation.utils';
import { Messages } from './PrototypePlaceholderPage.messages';

const fromSlug = (slug = '') =>
  slug.charAt(0).toUpperCase() + slug.slice(1).replace(/-/g, ' ');

// PMM Extensions apps that this build's side-car image leaves out.
const NOT_IN_THIS_BUILD = new Set([
  'data-archiving',
  'postgresql-backups',
  'replication-checksums',
  'schema-changes',
]);

// Stands in for a proposed sidebar entry or action that has no page yet
// (PMM-15353). The title comes from the entry that links here, or from the
// slug for an action button, so one page serves them all.
export const PrototypePlaceholderPage: FC = () => {
  const { navTree } = useNavigation();
  const { pathname } = useLocation();
  const { slug } = useParams();
  const title =
    findActiveNavItem(navTree, pathname)?.text ||
    fromSlug(slug) ||
    Messages.title;
  // A planned app's name is in the app header above, across the full width.
  const isApp = !!findAppPage(pathname);

  return (
    <Page title={title} hideTitle={isApp} maxWidth={isApp ? 'full' : undefined}>
      <Stack
        gap={2}
        sx={{ alignItems: 'flex-start' }}
        data-testid="prototype-placeholder"
      >
        <Chip
          size="small"
          color="info"
          variant="outlined"
          label={Messages.chip}
        />
        <Typography color="text.secondary">{Messages.description}</Typography>
        {slug && NOT_IN_THIS_BUILD.has(slug) && (
          <Typography color="text.secondary">
            {Messages.notInThisBuild}
          </Typography>
        )}
      </Stack>
    </Page>
  );
};
