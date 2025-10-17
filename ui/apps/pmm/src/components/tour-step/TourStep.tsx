import { Close } from '@mui/icons-material';
import { IconButton, Stack, Typography } from '@mui/material';
import { useTour } from '@reactour/tour';
import { FC, PropsWithChildren } from 'react';

interface Props extends PropsWithChildren {
  title: string;
}

const TourStep: FC<Props> = ({ title, children }) => {
  const { setIsOpen } = useTour();

  return (
    <Stack>
      <Stack direction="row" alignItems="center" justifyContent="space-between">
        <Typography variant="h5">{title}</Typography>
        <IconButton onClick={() => setIsOpen(false)}>
          <Close />
        </IconButton>
      </Stack>
      <Stack mt={2} gap={2}>
        {children}
      </Stack>
    </Stack>
  );
};

export default TourStep;
