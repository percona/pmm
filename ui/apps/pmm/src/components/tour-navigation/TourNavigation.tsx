import { FC } from 'react';
import { Messages } from './TourNavigation.messages';
import Stack from '@mui/material/Stack';
import Button from '@mui/material/Button';
import Box from '@mui/material/Box';
import ArrowBack from '@mui/icons-material/ArrowBack';
import ArrowForward from '@mui/icons-material/ArrowForward';

interface Props {
  currentStep: number;
  setCurrentStep: (step: number) => void;
  stepCount: number;
  endTour: () => void;
}

const TourNavigation: FC<Props> = ({
  currentStep,
  setCurrentStep,
  stepCount,
  endTour,
}) => {
  const isFirstStep = currentStep === 0;
  const isLastStep = currentStep === stepCount - 1;

  const nextStep = () => {
    if (currentStep < stepCount - 1) {
      setCurrentStep(currentStep + 1);
    }
  };

  const previousStep = () => {
    if (currentStep !== 0) {
      setCurrentStep(currentStep - 1);
    }
  };

  return (
    <Stack
      direction="row"
      justifyContent="space-between"
      alignItems="center"
      mt={4}
    >
      {!isFirstStep && (
        <Button
          data-testid="tour-previous-step-button"
          variant="text"
          onClick={previousStep}
          startIcon={<ArrowBack />}
        >
          {Messages.prev}
        </Button>
      )}
      <Box data-testid="tour-counter">
        {Messages.tip(currentStep + 1, stepCount)}
      </Box>
      {isLastStep ? (
        <Button
          data-testid="tour-end-tour-button"
          variant="contained"
          onClick={endTour}
        >
          {Messages.end}
        </Button>
      ) : (
        <Button
          data-testid="tour-next-step-button"
          variant="text"
          onClick={nextStep}
          endIcon={<ArrowForward />}
        >
          {Messages.next}
        </Button>
      )}
    </Stack>
  );
};

export default TourNavigation;
