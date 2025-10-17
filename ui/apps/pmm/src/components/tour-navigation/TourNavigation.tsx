import { ArrowBack, ArrowForward } from '@mui/icons-material';
import { Box, Button, Stack } from '@mui/material';
import { useTour } from '@reactour/tour';
import { FC } from 'react';
import { Messages } from './TourNavigation.messages';

const TourNavigation: FC = () => {
  const { currentStep, steps, setCurrentStep, setIsOpen } = useTour();

  return (
    <Stack
      direction="row"
      justifyContent="space-between"
      alignItems="center"
      mt={4}
    >
      {currentStep !== 0 && (
        <Button
          variant="text"
          onClick={() => setCurrentStep(currentStep - 1)}
          startIcon={<ArrowBack />}
        >
          {Messages.prev}
        </Button>
      )}
      <Box>{Messages.tip(currentStep + 1, steps.length)}</Box>
      {currentStep + 1 !== steps.length ? (
        <Button
          variant="text"
          onClick={() => setCurrentStep(currentStep + 1)}
          endIcon={<ArrowForward />}
        >
          {Messages.next}
        </Button>
      ) : (
        <Button variant="contained" onClick={() => setIsOpen(false)}>
          {Messages.end}
        </Button>
      )}
    </Stack>
  );
};

export default TourNavigation;
