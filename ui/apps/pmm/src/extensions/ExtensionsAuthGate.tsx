import { FC, PropsWithChildren, useEffect, useSyncExternalStore } from 'react';
import {
  Alert,
  AlertTitle,
  Box,
  Button,
  CircularProgress,
} from '@mui/material';
import { Messages } from './ExtensionsAuthGate.messages';
import {
  type ExtensionsAuthNotice,
  ensureExtensionsToken,
  getExtensionsAuthState,
  retryExtensionsAuth,
  subscribeExtensionsAuth,
} from './extensionsTokenStore';

const RetryButton: FC = () => (
  <Button
    color="inherit"
    size="small"
    sx={{ mr: 1 }}
    onClick={() => {
      void retryExtensionsAuth();
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
const ExtensionsAuthNoticeBar: FC<{ kind: ExtensionsAuthNotice }> = ({
  kind,
}) => (
  <Alert
    severity="warning"
    data-testid="extensions-auth-notice"
    action={<RetryButton />}
  >
    {kind === 'signedOut'
      ? Messages.notice.signedOut
      : Messages.notice.unreachable}
  </Alert>
);

/**
 * Holds a PMM Extensions route until a side-car bearer has been minted from the PMM session.
 *
 * Gating here rather than exchanging at app startup keeps the side-car out of the boot
 * path for the PMM users who never open a PMM Extensions page — the UI has no
 * `PMM_ENABLE_EXTENSIONS` flag to check, so an eager exchange would hit the side-car on every
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
export const ExtensionsAuthGate: FC<PropsWithChildren> = ({ children }) => {
  const { phase, notice } = useSyncExternalStore(
    subscribeExtensionsAuth,
    getExtensionsAuthState
  );

  useEffect(() => {
    // No-ops when a bearer is already held or the session was rejected; a
    // previous transient failure is retried on the next visit to a PMM Extensions route.
    void ensureExtensionsToken();
  }, []);

  if (phase === 'ready') {
    return (
      <>
        {notice !== null && <ExtensionsAuthNoticeBar kind={notice} />}
        {children}
      </>
    );
  }

  if (phase === 'signedOut' || phase === 'unreachable') {
    const signedOut = phase === 'signedOut';
    return (
      <Alert
        severity={signedOut ? 'warning' : 'error'}
        data-testid="extensions-auth-error"
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
