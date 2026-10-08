import { ReactNode } from 'react';
import { ButtonProps, MenuProps } from '@mui/material';

export type MenuButtonProps = {
  /** Renders the menu items; call the given `handleClose` in an item's onClick so the menu closes after a pick. */
  children?: (handleClose: () => void) => ReactNode;
  /** Label text shown in the button, to the left side of the dropdown arrow. */
  buttonText: string;
  /** MUI Button props forwarded to the trigger button, such as variant, size, or color. */
  buttonProps?: ButtonProps;
  /** MUI Menu props forwarded to the dropdown, for example to position it; leave `open` to the button. */
  menuProps?: MenuProps;
};
