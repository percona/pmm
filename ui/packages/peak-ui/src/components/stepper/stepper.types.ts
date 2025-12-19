import { StepperProps as MuiStepperProps } from '@mui/material/Stepper/Stepper';

export type StepperProps = MuiStepperProps & {
  noConnector?: boolean;
  dataTestId?: string;
};
