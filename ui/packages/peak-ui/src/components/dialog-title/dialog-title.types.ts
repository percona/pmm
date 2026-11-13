import { DialogTitleProps as MuiDialogTitleProps } from '@mui/material';

export type DialogTitleProps = {
  /** Called when the user clicks the close icon in the top-right corner; the icon only appears when this is set. */
  onClose?: () => void;
} & MuiDialogTitleProps;
