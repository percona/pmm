import { useState } from 'react';
import { Button, IconButton } from '@mui/material';
import Tooltip from '@mui/material/Tooltip';
import { ContentCopyOutlined as ContentCopyOutlinedIcon } from '@mui/icons-material';
import { CopyToClipboardButtonProps } from './CopyToClipboardButton.types';
import { Messages } from './clipboard.messages';

const CopyToClipboardButton = ({
  textToCopy,
  buttonProps,
  iconSx,
  showCopyButtonText,
  copyCommand = 'Copy command',
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
        <Button
          sx={{ ...buttonProps?.sx, display: 'flex', gap: 1 }}
          onClick={handleClick}
          disabled={!clipboardAvailable}
          {...buttonProps}
        >
          <ContentCopyOutlinedIcon sx={iconSx} />
          {copyCommand}
        </Button>
      ) : (
        <span>
          <IconButton
            component="div"
            sx={{
              ...buttonProps?.sx,
              '&.Mui-disabled': {
                pointerEvents: 'auto',
              },
            }}
            onClick={handleClick}
            disabled={!clipboardAvailable}
            {...buttonProps}
          >
            <ContentCopyOutlinedIcon sx={iconSx} />
          </IconButton>
        </span>
      )}
    </Tooltip>
  );
};

export default CopyToClipboardButton;
