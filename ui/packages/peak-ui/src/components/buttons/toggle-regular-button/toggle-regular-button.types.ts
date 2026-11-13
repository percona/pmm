import { ToggleButtonProps } from '@mui/material';

export type ToggleRegularButtonProps = {
  /** Label of the option. */
  children?: React.ReactNode;
  /** Identifier for automated tests. Defaults to "toggle-button-group-btn". */
  dataTestId?: string;
} & ToggleButtonProps;
