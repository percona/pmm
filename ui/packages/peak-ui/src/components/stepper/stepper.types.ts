import { StepperProps as MuiStepperProps } from '@mui/material';

export type StepperProps = MuiStepperProps & {
  noConnector?: boolean;
  dataTestId?: string;
};
