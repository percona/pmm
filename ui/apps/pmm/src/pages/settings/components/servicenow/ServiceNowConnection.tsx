import { FC, useEffect, useState } from 'react';
import Alert from '@mui/material/Alert';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import Stack from '@mui/material/Stack';
import { Messages } from '../../Settings.messages';
import { MAX_LABEL_WIDTH } from '../../Settings.constants';
import { SettingsFieldLabel } from '../settings-field-label';
import { useServiceNowConnection } from './ServiceNowConnection.hooks';
import { sepErrorMessage } from './ServiceNowConnection.utils';
import { ServiceNowConnected } from './ServiceNowConnected';
import { ServiceNowConnectionForm } from './ServiceNowConnectionForm';

/**
 * The ServiceNow connection tab: a heading that always renders, and one of five
 * bodies under it.
 *
 * The connection is optional, so its state is told by the body itself rather
 * than by a standing banner — a configured deployment sees what it has, an
 * unconfigured one sees how to get connected, and neither is asked to read a
 * status line to find out which.
 */
export const ServiceNowConnection: FC = () => {
  const {
    declaredNames,
    stored,
    status,
    isLoading,
    error: loadError,
    refetch,
  } = useServiceNowConnection();
  const [isRenewing, setIsRenewing] = useState(false);
  const { serviceNow } = Messages;

  // A disconnect elsewhere in the tab (or a drift arriving on a refetch) takes
  // the connected screen away on its own; the renewal flag would otherwise
  // survive it and hide the form's own reason for being open.
  useEffect(() => {
    if (status !== 'configured') {
      setIsRenewing(false);
    }
  }, [status]);

  const isConnected = status === 'configured' && !isRenewing;

  const body = () => {
    if (isLoading) {
      return (
        <Stack alignItems="center" py={4}>
          <CircularProgress
            aria-label={serviceNow.loading}
            data-testid="servicenow-loading"
          />
        </Stack>
      );
    }

    if (loadError) {
      return (
        <Alert
          severity="error"
          data-testid="servicenow-load-error"
          sx={{ maxWidth: MAX_LABEL_WIDTH }}
          action={
            <Button
              color="inherit"
              size="small"
              onClick={() => void refetch()}
              data-testid="servicenow-load-retry"
            >
              {serviceNow.retry}
            </Button>
          }
        >
          {sepErrorMessage(loadError, serviceNow.errors.loadFailed)}
        </Alert>
      );
    }

    // A SEP build that does not carry the key at all would answer any write with
    // a 422, so there is nothing to offer — as distinct from a deployment that
    // carries it and has simply not been configured yet.
    if (!stored.isPresent) {
      return (
        <Alert
          severity="info"
          data-testid="servicenow-unavailable"
          sx={{ maxWidth: MAX_LABEL_WIDTH }}
        >
          {serviceNow.unavailable}
        </Alert>
      );
    }

    if (isConnected) {
      return (
        <ServiceNowConnected
          stored={stored}
          onRenew={() => setIsRenewing(true)}
        />
      );
    }

    return (
      <ServiceNowConnectionForm
        declaredNames={declaredNames}
        stored={stored}
        isDrifted={status === 'drifted'}
        onCancel={isRenewing ? () => setIsRenewing(false) : undefined}
        onConnected={() => setIsRenewing(false)}
      />
    );
  };

  return (
    <Stack gap={4} sx={{ width: '640px' }}>
      {!isConnected && (
        <SettingsFieldLabel
          label={serviceNow.label}
          description={serviceNow.description}
          data-testid="servicenow-label"
        />
      )}
      {body()}
    </Stack>
  );
};
