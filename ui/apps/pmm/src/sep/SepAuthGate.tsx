import { FC, PropsWithChildren, useEffect, useSyncExternalStore } from 'react';
import {
  Alert,
  AlertTitle,
  Box,
  Button,
  CircularProgress,
} from '@mui/material';
import { Messages } from './SepAuthGate.messages';
import {
  ensureSepToken,
  getSepAuthStatus,
  retrySepAuth,
  subscribeSepAuth,
} from './sepTokenStore';

/**
 * Holds a SEP route until a SEP bearer has been minted from the PMM session.
 *
 * Gating here rather than exchanging at app startup keeps SEP out of the boot
 * path for the PMM users who never open a SEP page — the UI has no
 * `PMM_ENABLE_SEP` flag to check, so an eager exchange would hit SEP on every
 * page load for everybody.
 *
 * It also removes a race the token provider cannot: `setTokenProvider` is
 * synchronous, so a plugin's first queries would otherwise fire before the
 * exchange resolves and 401 on arrival. Children do not render until a bearer
 * is in hand.
 */
export const SepAuthGate: FC<PropsWithChildren> = ({ children }) => {
  const status = useSyncExternalStore(subscribeSepAuth, getSepAuthStatus);

  useEffect(() => {
    // No-ops when a bearer is already held or the session was rejected; a
    // previous transient failure is retried on the next visit to a SEP route.
    void ensureSepToken();
  }, []);

  if (status === 'ready') {
    return <>{children}</>;
  }

  if (status === 'signedOut' || status === 'error') {
    const signedOut = status === 'signedOut';
    return (
      <Alert
        severity={signedOut ? 'warning' : 'error'}
        data-testid="sep-auth-error"
        action={
          <Button
            color="inherit"
            size="small"
            onClick={() => {
              void retrySepAuth();
            }}
          >
            {Messages.retry}
          </Button>
        }
      >
        <AlertTitle>
          {signedOut ? Messages.signedOutTitle : Messages.errorTitle}
        </AlertTitle>
        {signedOut ? Messages.signedOut : Messages.error}
      </Alert>
    );
  }

  return (
    <Box
      sx={{
        flex: 1,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        p: 4,
      }}
    >
      <CircularProgress aria-label={Messages.loading} />
    </Box>
  );
};
