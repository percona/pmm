import { FC, useCallback, useEffect, useRef } from 'react';
import Alert from '@mui/material/Alert';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import Stack from '@mui/material/Stack';
import NetworkCheckIcon from '@mui/icons-material/NetworkCheck';
import { useConnectivityCheck } from '@sep/api';
import { Messages } from '../../Settings.messages';
import { DELIVERY_TARGETS } from './ServiceNowConnection.constants';
import {
  connectivityOutcome,
  deliveryResult,
  sepErrorMessage,
} from './ServiceNowConnection.utils';

interface Props {
  /**
   * Run the probe once on mount. Set right after a save, so "Verify and
   * connect" ends in a verdict instead of only a stored value.
   */
  autoRun?: boolean;
}

/**
 * Test what is stored against the receiver, without saving anything.
 *
 * The probe reads SEP's own stored delivery inputs server-side, so there is
 * nothing to send and nothing it can overwrite — an operator can ask whether
 * the current configuration works without first replacing it. That is also why
 * it is offered here and not on the form: credentials being typed are not
 * stored yet, so there would be nothing to probe with.
 *
 * Two failures are deliberately kept apart. A rejected call — 401, 403, a
 * network failure, a timeout on PMM's side — means the probe never ran, and is
 * reported as that. A `ConnectivityResult` that came back means it did run, and
 * is reported by its own status, whatever that status says.
 *
 * Admin-only comes for free: the settings page renders nothing to a non-admin
 * (`Page roles={[OrgRole.Admin]}`), and SEP holds the endpoint to `IsApiAdmin`
 * besides.
 */
export const ServiceNowConnectionTest: FC<Props> = ({ autoRun = false }) => {
  const { mutate, data, error, isPending } = useConnectivityCheck();
  const { serviceNow } = Messages;
  const hasAutoRun = useRef(false);

  const runTest = useCallback(
    () => mutate({ targets: DELIVERY_TARGETS }),
    [mutate]
  );

  useEffect(() => {
    // Guarded rather than keyed on `autoRun` alone: React re-runs effects on a
    // remount in development, and one save is one probe.
    if (autoRun && !hasAutoRun.current) {
      hasAutoRun.current = true;
      runTest();
    }
  }, [autoRun, runTest]);

  const result = deliveryResult(data);
  // The probe leaves the cluster and SEP bounds it at 15 seconds, so the
  // pending state has to stay legible for that long rather than flicker.
  const outcome = result ? connectivityOutcome(result) : undefined;

  return (
    <Stack gap={2} alignItems="flex-start">
      <Button
        variant="outlined"
        onClick={runTest}
        disabled={isPending}
        startIcon={
          isPending ? (
            <CircularProgress size={16} color="inherit" />
          ) : (
            <NetworkCheckIcon />
          )
        }
        data-testid="servicenow-test"
      >
        {isPending ? serviceNow.test.testing : serviceNow.test.action}
      </Button>

      {error ? (
        <Alert
          severity="error"
          sx={{ width: '100%' }}
          data-testid="servicenow-test-error"
        >
          {sepErrorMessage(error, serviceNow.errors.testFailed)}
        </Alert>
      ) : (
        outcome && (
          <Alert
            severity={outcome.severity}
            sx={{ width: '100%' }}
            data-testid="servicenow-test-result"
          >
            {outcome.message}
          </Alert>
        )
      )}
    </Stack>
  );
};
