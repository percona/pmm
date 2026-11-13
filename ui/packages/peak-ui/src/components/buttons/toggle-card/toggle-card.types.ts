import { ToggleButtonProps } from '@mui/material';

export type ToggleCardProps = {
  /** Content of the card: the option's title, description, or icon. */
  children?: React.ReactNode;
} & ToggleButtonProps;
