import { ButtonProps, SxProps, Theme } from '@mui/material';

export type CopyToClipboardButtonProps = {
  textToCopy: string;
  iconSx?: SxProps<Theme>;
  buttonProps?: ButtonProps;
  showCopyButtonText?: boolean;
  copyCommand?: string;
};
