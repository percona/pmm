import { Alert } from '@mui/material';
import { FC } from 'react';
import { LBAC_MESSAGE, logsErrorMessage } from '../utils/logsErrors';

export const LogsErrorAlert: FC<{ error: unknown }> = ({ error }) => {
  const message = logsErrorMessage(error);
  return (
    <Alert
      data-testid="logs-error"
      severity={message === LBAC_MESSAGE ? 'info' : 'error'}
    >
      {message}
    </Alert>
  );
};
