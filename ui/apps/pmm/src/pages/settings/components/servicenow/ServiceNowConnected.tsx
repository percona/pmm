import { FC, useState } from 'react';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { enqueueSnackbar } from 'notistack';
import { type ApiError, useResetSetting } from '@sep/api';
import { Modal } from 'components/modal';
import { Messages } from '../../Settings.messages';
import { MAX_LABEL_WIDTH } from '../../Settings.constants';
import {
  DELIVERY_INPUTS_KEY,
  SEP_SETTINGS_CLASS,
} from './ServiceNowConnection.constants';
import { StoredDeliveryInputs } from './ServiceNowConnection.types';
import {
  connectionIdentity,
  sepErrorMessage,
} from './ServiceNowConnection.utils';
import { ServiceNowConnectionTest } from './ServiceNowConnectionTest';

interface Props {
  stored: StoredDeliveryInputs;
  /** Whether this screen was reached by saving, rather than by loading. */
  justConnected?: boolean;
  onRenew: () => void;
}

const ConnectionDetail: FC<{
  label: string;
  value: string;
  testId: string;
}> = ({ label, value, testId }) => (
  <Stack>
    <Typography variant="caption" color="text.secondary">
      {label}
    </Typography>
    <Typography variant="body1" data-testid={testId}>
      {value}
    </Typography>
  </Stack>
);

/**
 * The connection as it stands, in place of the form that produced it.
 *
 * Only the endpoint is shown. PMM stores no author or timestamp for a saved
 * setting — SEP answers whether an override exists, not who wrote it — so the
 * rest of the design's detail row waits on a SEP endpoint that can answer it.
 * A blank stored endpoint means SEP is using the receiver its image bakes in,
 * which is the default this reports rather than an empty row.
 */
export const ServiceNowConnected: FC<Props> = ({
  stored,
  justConnected = false,
  onRenew,
}) => {
  const { mutateAsync: resetSetting, isPending: isDisconnecting } =
    useResetSetting();
  const [disconnectOpen, setDisconnectOpen] = useState(false);
  const { serviceNow } = Messages;

  const onDisconnect = async () => {
    try {
      await resetSetting({
        settingClass: SEP_SETTINGS_CLASS,
        key: DELIVERY_INPUTS_KEY,
      });
      enqueueSnackbar(serviceNow.disconnectSuccess, { variant: 'success' });
      setDisconnectOpen(false);
    } catch (error) {
      enqueueSnackbar(sepErrorMessage(error as ApiError), {
        variant: 'error',
      });
    }
  };

  return (
    <>
      <Stack
        gap={3}
        maxWidth={MAX_LABEL_WIDTH}
        data-testid="servicenow-connected"
      >
        <Typography variant="h6">{serviceNow.connectedTitle}</Typography>

        <ConnectionDetail
          label={serviceNow.endpointDetailLabel}
          value={stored.endpoint || serviceNow.defaultEndpoint}
          testId="servicenow-connected-endpoint"
        />

        <ServiceNowConnectionTest
          configurationId={connectionIdentity(stored)}
          autoRun={justConnected}
        />

        <Stack
          direction="row"
          alignItems="center"
          justifyContent="space-between"
          gap={2}
        >
          <Button
            variant="contained"
            onClick={onRenew}
            data-testid="servicenow-renew"
          >
            {serviceNow.renew}
          </Button>
          <Button
            variant="text"
            color="error"
            data-testid="servicenow-disconnect"
            onClick={() => setDisconnectOpen(true)}
          >
            {serviceNow.disconnect}
          </Button>
        </Stack>
      </Stack>

      <Modal
        open={disconnectOpen}
        onClose={() => setDisconnectOpen(false)}
        title={serviceNow.disconnectTitle}
      >
        <Stack gap={3}>
          <Typography variant="body2">{serviceNow.disconnectBody}</Typography>
          <Stack direction="row" gap={1} justifyContent="flex-end">
            <Button
              variant="text"
              onClick={() => setDisconnectOpen(false)}
              data-testid="servicenow-disconnect-cancel"
            >
              {serviceNow.disconnectCancel}
            </Button>
            <Button
              variant="contained"
              color="error"
              disabled={isDisconnecting}
              onClick={onDisconnect}
              data-testid="servicenow-disconnect-confirm"
            >
              {serviceNow.disconnectConfirm}
            </Button>
          </Stack>
        </Stack>
      </Modal>
    </>
  );
};
