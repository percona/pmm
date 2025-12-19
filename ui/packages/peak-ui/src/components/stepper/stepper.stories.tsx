import { Meta, StoryObj } from '@storybook/react';
import Stepper from './stepper';
import { StepperProps } from './stepper.types';
import {
  Box,
  Button,
  Step,
  StepButton,
  StepContent,
  StepLabel,
  Typography,
} from '@mui/material';
import React from 'react';
import * as DocBlock from '@storybook/blocks';

const steps = ['Step 1', 'Step 2', 'Step 3', 'Step 4'];

const meta = {
  title: 'Stepper',
  component: Stepper,
  tags: ['autodocs'],
  parameters: {
    layout: 'centered',
    docs: {
      toc: true,
      page: () => (
        <>
          <DocBlock.Title />
          <DocBlock.Subtitle />
          <DocBlock.Description />
          <DocBlock.Stories />
        </>
      ),
    },
  },
} satisfies Meta<StepperProps>;

export default meta;
type Story = StoryObj<Meta>;

export const BasicWithError: Story = {
  name: 'Stepper with error',
  render: function Render() {
    const isStepFailed = (step: number) => {
      return step === 1;
    };

    return (
      <Box
        sx={{
          width: '50rem',
          display: 'flex',
          flexDirection: 'column',
          gap: '5rem',
        }}
      >
        <Stepper activeStep={1}>
          {steps.map((label, index) => {
            const labelProps: {
              optional?: React.ReactNode;
              error?: boolean;
            } = {};
            if (isStepFailed(index)) {
              labelProps.optional = (
                <Typography variant="caption" color="error">
                  Alert message
                </Typography>
              );
              labelProps.error = true;
            }

            return (
              <Step key={label}>
                <StepLabel {...labelProps}> {label}</StepLabel>
              </Step>
            );
          })}
        </Stepper>
      </Box>
    );
  },
};

export const NonLinearStepper: Story = {
  name: 'Non-linear stepper',
  render: function Render() {
    const [activeStep, setActiveStep] = React.useState(0);
    const [verticalActiveStep, setVerticalActiveStep] = React.useState(0);
    const [completed, setCompleted] = React.useState<{
      [k: number]: boolean;
    }>({});
    const totalSteps = () => {
      return steps.length;
    };

    const completedSteps = () => {
      return Object.keys(completed).length;
    };

    const isLastStep = (stepper: 'vertical' | 'horizontal') => {
      return stepper === 'horizontal'
        ? activeStep === totalSteps() - 1
        : verticalActiveStep === totalSteps() - 1;
    };

    const allStepsCompleted = () => {
      return completedSteps() === totalSteps();
    };

    const handleNext = (stepper: 'vertical' | 'horizontal') => {
      let newActiveStep: number;
      if (isLastStep(stepper) && !allStepsCompleted()) {
        newActiveStep = steps.findIndex((step, i) => !(i in completed));
      } else {
        newActiveStep =
          stepper === 'horizontal' ? activeStep + 1 : verticalActiveStep + 1;
      }
      stepper === 'horizontal'
        ? setActiveStep(newActiveStep)
        : setVerticalActiveStep(newActiveStep);
    };
    const handleBack = (stepper: 'vertical' | 'horizontal') => {
      stepper === 'horizontal'
        ? setActiveStep((prevActiveStep) => prevActiveStep - 1)
        : setVerticalActiveStep(
            (prevVerticalActiveStep) => prevVerticalActiveStep - 1
          );
    };

    const handleStep = (step: number) => () => {
      setActiveStep(step);
    };

    const handleComplete = (stepper: 'vertical' | 'horizontal') => {
      const newCompleted = completed;
      newCompleted[activeStep] = true;
      setCompleted(newCompleted);
      handleNext(stepper);
    };

    const handleReset = () => {
      setActiveStep(0);
      setCompleted({});
    };

    return (
      <Box
        sx={{
          width: '50rem',
          display: 'flex',
          flexDirection: 'column',
          gap: '5rem',
        }}
      >
        <Stepper nonLinear activeStep={activeStep}>
          {steps.map((label, index) => (
            <Step key={label} completed={completed[index]}>
              <StepButton color="inherit" onClick={handleStep(index)}>
                {label}
              </StepButton>
            </Step>
          ))}
        </Stepper>
        <div>
          {allStepsCompleted() ? (
            <React.Fragment>
              <Typography sx={{ mt: 2, mb: 1 }}>
                All steps completed - you&apos;re finished
              </Typography>
              <Box sx={{ display: 'flex', flexDirection: 'row', pt: 2 }}>
                <Box sx={{ flex: '1 1 auto' }} />
                <Button onClick={handleReset}>Reset</Button>
              </Box>
            </React.Fragment>
          ) : (
            <React.Fragment>
              <Typography sx={{ mt: 2, mb: 1, py: 1 }}>
                Step {activeStep + 1}
              </Typography>
              <Box sx={{ display: 'flex', flexDirection: 'row', pt: 2 }}>
                <Button
                  color="inherit"
                  disabled={activeStep === 0}
                  onClick={() => handleBack('horizontal')}
                  sx={{ mr: 1 }}
                >
                  Back
                </Button>
                <Box sx={{ flex: '1 1 auto' }} />
                <Button onClick={() => handleNext('horizontal')} sx={{ mr: 1 }}>
                  Next
                </Button>
                {activeStep !== steps.length &&
                  (completed[activeStep] ? (
                    <Typography
                      variant="caption"
                      sx={{ display: 'inline-block' }}
                    >
                      Step {activeStep + 1} already completed
                    </Typography>
                  ) : (
                    <Button onClick={() => handleComplete('horizontal')}>
                      {completedSteps() === totalSteps() - 1
                        ? 'Finish'
                        : 'Complete Step'}
                    </Button>
                  ))}
              </Box>
            </React.Fragment>
          )}
        </div>
      </Box>
    );
  },
};

export const VerticalStepper: Story = {
  name: 'Vertical stepper',
  render: function Render() {
    const [activeStep, setActiveStep] = React.useState(0);
    const [verticalActiveStep, setVerticalActiveStep] = React.useState(0);
    const [completed] = React.useState<{
      [k: number]: boolean;
    }>({});
    const totalSteps = () => {
      return steps.length;
    };

    const completedSteps = () => {
      return Object.keys(completed).length;
    };

    const isLastStep = (stepper: 'vertical' | 'horizontal') => {
      return stepper === 'horizontal'
        ? activeStep === totalSteps() - 1
        : verticalActiveStep === totalSteps() - 1;
    };

    const allStepsCompleted = () => {
      return completedSteps() === totalSteps();
    };

    const handleNext = (stepper: 'vertical' | 'horizontal') => {
      let newActiveStep: number;
      if (isLastStep(stepper) && !allStepsCompleted()) {
        newActiveStep = steps.findIndex((step, i) => !(i in completed));
      } else {
        newActiveStep =
          stepper === 'horizontal' ? activeStep + 1 : verticalActiveStep + 1;
      }
      stepper === 'horizontal'
        ? setActiveStep(newActiveStep)
        : setVerticalActiveStep(newActiveStep);
    };
    const handleBack = (stepper: 'vertical' | 'horizontal') => {
      stepper === 'horizontal'
        ? setActiveStep((prevActiveStep) => prevActiveStep - 1)
        : setVerticalActiveStep(
            (prevVerticalActiveStep) => prevVerticalActiveStep - 1
          );
    };

    return (
      <Box
        sx={{
          width: '50rem',
          display: 'flex',
          flexDirection: 'column',
          gap: '5rem',
        }}
      >
        <Stepper activeStep={verticalActiveStep} orientation="vertical">
          {steps.map((step, index) => (
            <Step key={index}>
              <StepLabel>{step}</StepLabel>
              <StepContent>
                <Typography>{step} Text</Typography>
                <Box sx={{ mb: 2 }}>
                  <div>
                    <Button
                      variant="contained"
                      onClick={() => handleNext('vertical')}
                      sx={{ mt: 1, mr: 1 }}
                    >
                      {index === steps.length - 1 ? 'Finish' : 'Continue'}
                    </Button>
                    <Button
                      disabled={index === 0}
                      onClick={() => handleBack('vertical')}
                      sx={{ mt: 1, mr: 1 }}
                    >
                      Back
                    </Button>
                  </div>
                </Box>
              </StepContent>
            </Step>
          ))}
        </Stepper>
      </Box>
    );
  },
};
