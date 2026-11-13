import { DialogProps as MuiDialogProps } from '@mui/material';

export type DialogProps = {
  /** Shows a progress bar along the top edge while an action runs; the content stays visible. */
  loading?: boolean;
} & MuiDialogProps;
