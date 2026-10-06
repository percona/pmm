import { ButtonProps, SxProps, Theme } from '@mui/material';

export type CopyToClipboardButtonProps = {
  /** Text placed on the user's clipboard when the button is clicked. */
  textToCopy: string;
  /** Styles for the copy icon, for example to resize or recolor it. */
  iconSx?: SxProps<Theme>;
  /** MUI Button props forwarded to the underlying button, such as size, color, or sx. */
  buttonProps?: ButtonProps;
  /** Shows a labeled button instead of the bare icon; the label is `copyCommand`. */
  showCopyButtonText?: boolean;
  /** Label of the labeled button. Defaults to "Copy code". */
  copyCommand?: string;
};
