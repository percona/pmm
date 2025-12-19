import { forwardRef } from 'react';
import { CustomContentProps } from 'notistack';
import { Alert } from '@mui/material';

const NotistackMuiSnackbar = forwardRef<HTMLDivElement, CustomContentProps>(
  (props, ref) => {
    const { message, variant } = props;

    return variant !== 'default' ? (
      <Alert ref={ref} severity={variant}>
        {message}
      </Alert>
    ) : (
      message
    );
  }
);

export default NotistackMuiSnackbar;
