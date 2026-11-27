import type { FC, PropsWithChildren } from 'react';
import { useParams } from 'react-router-dom';
import Alert from '@mui/material/Alert';
import AlertTitle from '@mui/material/AlertTitle';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import Stack from '@mui/material/Stack';
import { useExtensionsApps } from 'hooks/api/useExtensionsApps';
import { Messages } from './ExtensionsAppGuard.messages';

interface ExtensionsAppGuardProps {
  /** The app a dedicated route serves; without it, the app `:appSlug` names. */
  appKey?: string;
}

/**
 * Decides from the side-car's app list whether an app's page may mount: a
 * plugin reports a schema error as a connection problem, which would mislead
 * for an unknown or disabled app.
 *
 * A dedicated route mounts its app whenever the list cannot say otherwise:
 * when the list cannot be loaded, or does not include the app. The generic
 * route has nothing to render without the list, and renders only listed apps
 * without a page of their own.
 */
export const ExtensionsAppGuard: FC<
  PropsWithChildren<ExtensionsAppGuardProps>
> = ({ appKey, children }) => {
  const { appSlug } = useParams();
  const { data: apps, isError, refetch } = useExtensionsApps();

  // A failed refetch keeps the list it already has, so only a first load can fail.
  if (!apps && isError) {
    if (appKey) {
      return children;
    }

    return (
      <Alert
        severity="error"
        data-testid="extensions-app-load-failed"
        action={
          <Button color="inherit" size="small" onClick={() => refetch()}>
            {Messages.retry}
          </Button>
        }
      >
        <AlertTitle>{Messages.loadFailedTitle}</AlertTitle>
        {Messages.loadFailed}
      </Alert>
    );
  }

  if (!apps) {
    return (
      <Stack alignItems="center" py={4}>
        <CircularProgress aria-label={Messages.loading} />
      </Stack>
    );
  }

  const app = apps.find((candidate) =>
    appKey ? candidate.key === appKey : candidate.slug === appSlug
  );

  if (!app && appKey) {
    return children;
  }

  if (!app || (!appKey && app.customUi)) {
    return (
      <Alert severity="warning" data-testid="extensions-app-not-found">
        <AlertTitle>{Messages.notFoundTitle}</AlertTitle>
        {Messages.notFound}
      </Alert>
    );
  }

  if (!app.enabled) {
    return (
      <Alert severity="info" data-testid="extensions-app-disabled">
        <AlertTitle>{Messages.disabledTitle(app.label)}</AlertTitle>
        {app.blockedBy.length
          ? Messages.blockedBy(app.blockedBy)
          : Messages.disabled}
      </Alert>
    );
  }

  return children;
};
