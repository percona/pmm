import { ReactNode } from 'react';
import { ButtonProps, MenuProps } from '@mui/material';

export type MenuButtonProps = {
  children?: (handleClose: () => void) => ReactNode;
  buttonText: string;
  buttonProps?: ButtonProps;
  menuProps?: MenuProps;
};
