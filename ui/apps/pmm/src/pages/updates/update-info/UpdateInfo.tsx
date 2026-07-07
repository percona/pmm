import { Link, Stack, Typography } from '@mui/material';
import { FC } from 'react';
import { Messages } from './UpdateInfo.messages';
import { PMM_DOCS_UPDATES_URL } from 'lib/constants';

export const UpdateInfo: FC = () => (
  <Stack
    spacing={3}
    sx={{
      mt: 3,
    }}
  >
    <Stack spacing={1}>
      <Typography variant="h5">{Messages.title}</Typography>
      <Typography>
        {Messages.upgrading}
      </Typography>
    </Stack>
    <Stack spacing={1}>
      <Typography variant="sectionHeading" textTransform="uppercase">
        {Messages.whatsNext}
      </Typography>
      <Typography>{Messages.afterCompleting}</Typography>
    </Stack>
  </Stack>
);
