import Close from '@mui/icons-material/Close';
import { Box, IconButton } from '@mui/material';
import { FC } from 'react';

interface Props {
  endTour: () => void;
}

const TourCloseButton: FC<Props> = ({ endTour }) => (
  <Box
    sx={(theme) => ({
      position: 'absolute',
      top: theme.spacing(1),
      right: theme.spacing(1),
    })}
  >
    <IconButton data-testid="tour-close-button" onClick={endTour}>
      <Close />
    </IconButton>
  </Box>
);

export default TourCloseButton;
