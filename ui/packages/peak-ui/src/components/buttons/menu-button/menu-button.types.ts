import { ReactNode } from 'react';
import { ButtonProps, MenuProps } from '@mui/material';

export type MenuButtonProps = {
  /** Renders the menu items; call the given `handleClose` in an item's onClick so the menu closes after a pick. */
  children?: (handleClose: () => void) => ReactNode;
  /** Label text shown in the button, left of the dropdown arrow. */
  buttonText: string;
  /** MUI Button props forwarded to the trigger button, such as variant, size, or color. */
  buttonProps?: ButtonProps;
  /** MUI Menu props for the dropdown; they take over its open state, and a `MenuListProps` here drops the menu's link to the button. */
  menuProps?: MenuProps;
};
