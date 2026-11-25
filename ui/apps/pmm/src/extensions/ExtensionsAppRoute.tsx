import type { FC } from 'react';
import { useParams } from 'react-router-dom';
import Alert from '@mui/material/Alert';
import AlertTitle from '@mui/material/AlertTitle';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import Stack from '@mui/material/Stack';
import { SchemaDrivenPlugin } from '@pmm-extensions/framework';
import { useExtensionsApps } from 'hooks/api/useExtensionsApps';
import { extensionsAppPath } from './extensionsApps';
import { Messages } from './ExtensionsAppRoute.messages';

/**
 * Renders any schema-driven app the side-car lists, by slug. Apps that need
 * more than the generic page keep a dedicated route, which the router ranks
 * above this one.
 *
 * Whether the slug names an app, and whether that app is enabled, is decided
 * from the list before the plugin mounts: the plugin reports a schema error
 * as a connection problem, which would mislead for an unknown or disabled app.
 */
export const ExtensionsAppRoute: FC = () => {
  const { appSlug } = useParams();
  const { data: apps, isError, refetch } = useExtensionsApps();

  // A failed refetch keeps the list it already has, so only a first load can fail.
  if (!apps && isError) {
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

  const app = apps.find((candidate) => candidate.slug === appSlug);

  if (!app || app.customUi) {
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

  return (
    <SchemaDrivenPlugin
      // A fresh plugin per app, so no state carries over between two apps.
      key={app.key}
      pluginName={app.key}
      routeBase={extensionsAppPath(app.slug)}
    />
  );
};
