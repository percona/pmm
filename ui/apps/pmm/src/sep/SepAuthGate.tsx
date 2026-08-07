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
  type SepAuthNotice,
  ensureSepToken,
  getSepAuthState,
  retrySepAuth,
  subscribeSepAuth,
} from './sepTokenStore';

const RetryButton: FC = () => (
  <Button
    color="inherit"
    size="small"
    onClick={() => {
      void retrySepAuth();
    }}
  >
    {Messages.retry}
  </Button>
);

/**
 * Inline report of a failure that arrived after the page was already open.
 *
 * Deliberately not a replacement for the page: a background renewal failing
 * must not discard a half-filled form. It tells the user that submitting will
 * fail and offers a retry, and leaves everything else alone.
 */
const SepAuthNoticeBar: FC<{ kind: SepAuthNotice }> = ({ kind }) => (
  <Alert
    severity="warning"
    data-testid="sep-auth-notice"
    action={<RetryButton />}
  >
    {kind === 'signedOut'
      ? Messages.notice.signedOut
      : Messages.notice.unreachable}
  </Alert>
);

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
 *
 * Once they have rendered they stay rendered. A later failure is reported by
 * `notice`, beside the page rather than instead of it.
 */
export const SepAuthGate: FC<PropsWithChildren> = ({ children }) => {
  const { phase, notice } = useSyncExternalStore(
    subscribeSepAuth,
    getSepAuthState
  );

  useEffect(() => {
    // No-ops when a bearer is already held or the session was rejected; a
    // previous transient failure is retried on the next visit to a SEP route.
    void ensureSepToken();
  }, []);

  if (phase === 'ready') {
    return (
      <>
        {notice !== null && <SepAuthNoticeBar kind={notice} />}
        {children}
      </>
    );
  }

  if (phase === 'signedOut' || phase === 'unreachable') {
    const signedOut = phase === 'signedOut';
    return (
      <Alert
        severity={signedOut ? 'warning' : 'error'}
        data-testid="sep-auth-error"
        action={<RetryButton />}
      >
        <AlertTitle>
          {signedOut
            ? Messages.blocked.signedOutTitle
            : Messages.blocked.unreachableTitle}
        </AlertTitle>
        {signedOut ? Messages.blocked.signedOut : Messages.blocked.unreachable}
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
