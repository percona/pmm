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
   * Identity of the configuration a verdict would describe. When it changes,
   * any verdict on screen described something else and is dropped.
   */
  configurationId: string;
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
 * A verdict never outlives what it described. It says something about the
 * configuration that was stored when the probe ran, so a configuration that
 * changes underneath it — saved from another session, or arriving on a
 * refetch — takes it away rather than leaving it standing beside a receiver it
 * never reached.
 *
 * Admin-only comes for free: the settings page renders nothing to a non-admin
 * (`Page roles={[OrgRole.Admin]}`), and SEP holds the endpoint to `IsApiAdmin`
 * besides.
 */
export const ServiceNowConnectionTest: FC<Props> = ({
  configurationId,
  autoRun = false,
}) => {
  const { mutate, reset, data, error, isPending } = useConnectivityCheck();
  const { serviceNow } = Messages;
  const hasAutoRun = useRef(false);
  const probedConfiguration = useRef(configurationId);

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

  useEffect(() => {
    if (probedConfiguration.current !== configurationId) {
      probedConfiguration.current = configurationId;
      // Deliberately not a re-probe: the operator asked about the configuration
      // that was there before, and probing something they have not looked at
      // yet would answer a question nobody asked.
      //
      // This relies on `reset()` detaching the observer from a probe still in
      // flight, so its verdict cannot land against the configuration that
      // replaced the one it was asked about. If that ever stops holding, this
      // needs to track which configuration each run was started for rather
      // than only the boundary at which it changed.
      reset();
    }
  }, [configurationId, reset]);

  // A re-test keeps the previous `data` until the new call resolves, so the
  // verdict is withheld while one is in flight rather than answering for the
  // run the operator is currently waiting on. The probe leaves the cluster and
  // SEP bounds it at 15 seconds, so that wait has to stay legible for that
  // long rather than flicker.
  const result = isPending ? undefined : deliveryResult(data);
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

      {error && !isPending ? (
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
