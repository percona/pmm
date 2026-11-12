import { useState } from 'react';
import { render } from '@testing-library/react';
import { Box, Button, Step, StepLabel, Typography } from '@mui/material';
import Stepper from '.';
import { baseThemeOptions, ThemeContextProvider } from '../../design';

const StepperWithoutConnectors = () => {
  const steps = [
    { label: 'Step title' },
    { label: 'Step title' },
    { label: 'Step title' },
    { label: 'Step title' },
    { label: 'Step title' },
    { label: 'Step title' },
  ];
  const [activeStep, setActiveStep] = useState(0);
  const [skipped, setSkipped] = useState(new Set<number>());

  const isStepSkipped = (step: number) => {
    return skipped.has(step);
  };

  const handleNext = () => {
    let newSkipped = skipped;
    if (isStepSkipped(activeStep)) {
      newSkipped = new Set(newSkipped.values());
      newSkipped.delete(activeStep);
    }

    setActiveStep((prevActiveStep) => prevActiveStep + 1);
    setSkipped(newSkipped);
  };

  const handleBack = () => {
    setActiveStep((prevActiveStep) => prevActiveStep - 1);
  };

  const handleReset = () => {
    setActiveStep(0);
  };

  return (
    <ThemeContextProvider themeOptions={baseThemeOptions}>
      <Box sx={{ padding: 1 }}>
        <Stepper activeStep={activeStep} noConnector dataTestId="noConnector">
          {steps.map(({ label }, index) => {
            const stepProps: { completed?: boolean } = {};
            return (
              <Step key={`${label}_${index}`} {...stepProps}>
                <StepLabel />
              </Step>
            );
          })}
        </Stepper>
        {activeStep === steps.length ? (
          <>
            <Typography sx={{ mt: 2, mb: 1 }}>
              All steps completed - you&apos;re finished
            </Typography>
            <Box sx={{ display: 'flex', flexDirection: 'row', pt: 2 }}>
              <Box sx={{ flex: '1 1 auto' }} />
              <Button onClick={handleReset}>Reset</Button>
            </Box>
          </>
        ) : (
          <>
            <Box sx={{ display: 'flex', flexDirection: 'row', pt: 2 }}>
              <Button
                color="inherit"
                disabled={activeStep === 0}
                onClick={handleBack}
                sx={{ mr: 1 }}
              >
                Back
              </Button>
              <Box sx={{ flex: '1 1 auto' }} />
              <Button onClick={handleNext}>
                {activeStep === steps.length - 1 ? 'Finish' : 'Next'}
              </Button>
            </Box>
          </>
        )}
      </Box>
    </ThemeContextProvider>
  );
};
const HorizontalStepperWithConnectors = () => {
  const steps = [
    { label: 'Step 1 title' },
    { label: 'Step 2 title' },
    { label: 'Step 3 title' },
  ];
  const [activeStep, setActiveStep] = useState(0);
  const [skipped, setSkipped] = useState(new Set<number>());

  const handleNext = () => {
    setActiveStep((prevActiveStep) => prevActiveStep + 1);
    setSkipped(skipped);
  };

  const handleBack = () => {
    setActiveStep((prevActiveStep) => prevActiveStep - 1);
  };

  const handleReset = () => {
    setActiveStep(0);
  };

  return (
    <ThemeContextProvider themeOptions={baseThemeOptions}>
      <Box sx={{ padding: 1, width: '400px' }}>
        <Stepper activeStep={activeStep}>
          {steps.map(({ label }) => {
            const stepProps: { completed?: boolean } = {};
            const labelProps: {
              optional?: React.ReactNode;
            } = {};
            return (
              <Step key={label} {...stepProps}>
                <StepLabel {...labelProps}>{label}</StepLabel>
              </Step>
            );
          })}
        </Stepper>
        {activeStep === steps.length ? (
          <>
            <Box sx={{ display: 'flex', flexDirection: 'row', pt: 2 }}>
              <Box sx={{ flex: '1 1 auto' }} />
              <Button onClick={handleReset}>Reset</Button>
            </Box>
          </>
        ) : (
          <>
            <Box sx={{ display: 'flex', flexDirection: 'row', pt: 2 }}>
              <Button
                color="inherit"
                disabled={activeStep === 0}
                onClick={handleBack}
                sx={{ mr: 1 }}
              >
                Back
              </Button>
              <Box sx={{ flex: '1 1 auto' }} />
              <Button onClick={handleNext}>
                {activeStep === steps.length - 1 ? 'Finish' : 'Next'}
              </Button>
            </Box>
          </>
        )}
      </Box>
    </ThemeContextProvider>
  );
};

describe('Stepper', () => {
  it('should not show the connector line when noConnector prop is true', () => {
    const { container } = render(<StepperWithoutConnectors />);
    expect(
      container.getElementsByClassName('MuiStepConnector-root')[0]
    ).toBeUndefined();
  });
  it('should show the connector line when noConnector prop is false', () => {
    const { container } = render(<HorizontalStepperWithConnectors />);
    expect(
      container.getElementsByClassName('MuiStepConnector-root')[0]
    ).toBeTruthy();
  });
});
