import type { FC } from 'react';
import { useParams } from 'react-router-dom';
import { SchemaDrivenPlugin } from '@pmm-extensions/framework';
import { useExtensionsApps } from 'hooks/api/useExtensionsApps';
import { extensionsAppPath } from './extensionsApps';

/**
 * Renders any schema-driven app the side-car lists, by slug. Apps that need
 * more than the generic page keep a dedicated route, which the router ranks
 * above this one. Mounted inside `ExtensionsAppGuard`, which has already
 * decided that the slug names an enabled app.
 */
export const ExtensionsAppRoute: FC = () => {
  const { appSlug } = useParams();
  const { data: apps } = useExtensionsApps();
  const app = apps?.find((candidate) => candidate.slug === appSlug);

  if (!app) {
    return null;
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
