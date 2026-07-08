import { useState } from 'react';
import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import ContentCopyOutlinedIcon from '@mui/icons-material/ContentCopyOutlined';
import { CopyToClipboardButtonProps } from './CopyToClipboardButton.types';
import { Messages } from './clipboard.messages';
import { mergeSx } from '@/utils';

const CopyToClipboardButton = ({
  textToCopy,
  buttonProps,
  iconSx,
  showCopyButtonText,
  copyCommand = 'Copy code',
}: CopyToClipboardButtonProps) => {
  const [open, setOpen] = useState(false);
  const clipboardAvailable = !!navigator.clipboard;

  const handleClick = () => {
    if (clipboardAvailable) {
      navigator.clipboard.writeText(textToCopy);
      setOpen(true);

      setTimeout(() => {
        setOpen(false);
      }, 1500);
    }
  };

  return (
    <Tooltip
      {...(clipboardAvailable && {
        disableHoverListener: true,
        open,
      })}
      disableFocusListener
      disableTouchListener
      PopperProps={{
        disablePortal: true,
      }}
      title={clipboardAvailable ? Messages.copied : Messages.restrictedAccess}
    >
      {showCopyButtonText ? (
        <span>
          <Button
            startIcon={<ContentCopyOutlinedIcon sx={iconSx} />}
            onClick={handleClick}
            disabled={!clipboardAvailable}
            {...buttonProps}
          >
            {copyCommand}
          </Button>
        </span>
      ) : (
        <span>
          <IconButton
            component="div"
            onClick={handleClick}
            disabled={!clipboardAvailable}
            {...buttonProps}
            sx={mergeSx({ '&.Mui-disabled': { pointerEvents: 'auto' } }, buttonProps?.sx)}
          >
            <ContentCopyOutlinedIcon sx={iconSx} />
          </IconButton>
        </span>
      )}
    </Tooltip>
  );
};

export default CopyToClipboardButton;
