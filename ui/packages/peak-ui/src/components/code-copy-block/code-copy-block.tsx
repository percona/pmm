import { Alert } from '@mui/material';
import { CodeCopyBlockProps } from './code-copy-block.types';
import CopyToClipboardButton from '../buttons/copy-to-clipboard-button';

const CodeCopyBlock = ({ message, showCopyButtonText }: CodeCopyBlockProps) => {
  return (
    <Alert
      severity="info"
      icon={false}
      sx={{
        mt: 0.5,
        mb: 0.5,
        '& .MuiAlert-action': {
          alignItems: 'center',
          pt: 0,
        },
        fontFamily: '"Roboto Mono", "Helvetica", "Arial", "sans-serif"',
      }}
      action={
        <CopyToClipboardButton
          showCopyButtonText={showCopyButtonText}
          buttonProps={{ size: 'small', color: 'primary' }}
          textToCopy={message}
        />
      }
    >
      {message}
    </Alert>
  );
};

export default CodeCopyBlock;
