import { Link, Stack, Typography } from '@mui/material';
import { FC } from 'react';
import { Messages } from './UpdateInfo.messages';

export const UpdateInfo: FC = () => (
  <Stack
    spacing={1}
    sx={{
      mt: 3,
    }}
  >
    <Typography variant="h5">{Messages.title}</Typography>
    <Typography>
      {Messages.upgrading}
      <br />
      <Link>{Messages.readMore}</Link>
    </Typography>
    <Typography variant="sectionHeading" textTransform="uppercase">
      {Messages.whatsNext}
    </Typography>
    <Typography>{Messages.afterCompleting}</Typography>
  </Stack>
);
