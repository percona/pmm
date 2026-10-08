import { FC } from 'react';
import { useLocation } from 'react-router-dom';
import Chip from '@mui/material/Chip';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Page } from 'components/page';
import { useNavigation } from 'contexts/navigation';
import { findActiveNavItem } from 'utils/navigation.utils';
import { Messages } from './PrototypePlaceholderPage.messages';

// Stands in for a proposed sidebar entry that has no page yet (PMM-15353). The
// title comes from the entry that links here, so one page serves them all.
export const PrototypePlaceholderPage: FC = () => {
  const { navTree } = useNavigation();
  const { pathname } = useLocation();
  const title = findActiveNavItem(navTree, pathname)?.text || Messages.title;

  return (
    <Page title={title}>
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
      </Stack>
    </Page>
  );
};
