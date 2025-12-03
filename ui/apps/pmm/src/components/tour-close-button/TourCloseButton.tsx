import Close from '@mui/icons-material/Close';
import Box from '@mui/material/Box';
import IconButton from '@mui/material/IconButton';
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
