import { Stack, Typography } from '@mui/material';
import { FC, PropsWithChildren } from 'react';

interface Props extends PropsWithChildren {
  title: string;
}

const TourStep: FC<Props> = ({ title, children }) => (
  <Stack>
    <Typography data-testid="tour-step-title" variant="h5">
      {title}
    </Typography>
    <Stack data-testid="tour-step-body" mt={2} gap={2}>
      {children}
    </Stack>
  </Stack>
);

export default TourStep;
