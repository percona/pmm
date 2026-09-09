import { FC, useState } from 'react';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { enqueueSnackbar } from 'notistack';
import { type ApiError, useResetSetting } from '@sep/api';
import { Modal } from 'components/modal';
import { Messages } from '../../Settings.messages';
import {
  DELIVERY_INPUTS_KEY,
  SEP_SETTINGS_CLASS,
} from './ServiceNowConnection.constants';
import { sepErrorMessage } from './ServiceNowConnection.utils';

interface Props {
  /**
   * Shown above the button where the screen is not otherwise about the stored
   * connection — the form has just asked for credentials, so a bare Disconnect
   * there reads as removing what the operator is in the middle of typing.
   */
  hint?: string;
}

/**
 * Clearing the stored delivery inputs, behind a confirmation.
 *
 * It renders wherever SEP holds an override, not only where that override
 * satisfies the delivery plan: an override the plan no longer accepts is the
 * state most in need of removing, and Disconnect is the only route that removes
 * one now that a blank save is refused.
 */
export const ServiceNowDisconnect: FC<Props> = ({ hint }) => {
  const { mutateAsync: resetSetting, isPending: isDisconnecting } =
    useResetSetting();
  const [isOpen, setIsOpen] = useState(false);
  const { serviceNow } = Messages;

  const onDisconnect = async () => {
    try {
      await resetSetting({
        settingClass: SEP_SETTINGS_CLASS,
        key: DELIVERY_INPUTS_KEY,
      });
      enqueueSnackbar(serviceNow.disconnectSuccess, { variant: 'success' });
      setIsOpen(false);
    } catch (error) {
      enqueueSnackbar(sepErrorMessage(error as ApiError), {
        variant: 'error',
      });
    }
  };

  return (
    <>
      <Stack gap={0.5} alignItems="flex-start">
        {hint && (
          <Typography variant="caption" color="text.secondary">
            {hint}
          </Typography>
        )}
        <Button
          variant="text"
          color="error"
          data-testid="servicenow-disconnect"
          onClick={() => setIsOpen(true)}
        >
          {serviceNow.disconnect}
        </Button>
      </Stack>

      <Modal
        open={isOpen}
        onClose={() => setIsOpen(false)}
        title={serviceNow.disconnectTitle}
      >
        <Stack gap={3}>
          <Typography variant="body2">{serviceNow.disconnectBody}</Typography>
          <Stack direction="row" gap={1} justifyContent="flex-end">
            <Button
              variant="text"
              onClick={() => setIsOpen(false)}
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
