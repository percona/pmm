import { StepperProps as MuiStepperProps } from '@mui/material';

export type StepperProps = MuiStepperProps & {
  /** Removes the lines between steps and their spacing, for compact wizards. */
  noConnector?: boolean;
  /** Identifier for automated tests. */
  dataTestId?: string;
};
