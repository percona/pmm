import AddOutlinedIcon from '@mui/icons-material/AddOutlined';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { NothingFoundIllustration } from '@percona/peak-ui';
import { PMM_DUMP_NEW_PATH } from 'lib/constants';
import type { FC } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { Messages } from '../../DumpPage.messages';

export const DumpEmptyState: FC = () => (
  <Stack
    gap={3}
    sx={{
      py: 6,
      mx: 'auto',
      maxWidth: 480,
      alignItems: 'center',
      textAlign: 'center',
    }}
    data-testid="dump-empty-state"
  >
    <NothingFoundIllustration
      color="primary"
      aria-hidden
      sx={{ height: 160, width: 160, mb: -3 }}
    />
    <Stack gap={1}>
      <Typography variant="h3" component="h2">
        {Messages.emptyTitle}
      </Typography>
      <Typography variant="body1" color="text.secondary">
        {Messages.emptyDescription}
      </Typography>
    </Stack>
    <Button
      variant="contained"
      component={RouterLink}
      to={PMM_DUMP_NEW_PATH}
      startIcon={<AddOutlinedIcon />}
      data-testid="create-dataset"
    >
      {Messages.createDataset}
    </Button>
  </Stack>
);
