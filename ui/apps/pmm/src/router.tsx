import React from 'react';
import { Navigate, createBrowserRouter, useLocation } from 'react-router-dom';
import { Settings } from 'pages/settings';
import { Updates } from 'pages/updates';
import { UpdateClients } from 'pages/update-clients/UpdateClients';
import { MainWithNav } from 'components/main/MainWithNav';
import { NotFoundPage } from 'pages/not-found';
import { HelpCenter } from 'pages/help-center';
import { RealtimeSelection } from 'pages/rta/selection';
import Providers from 'Providers';
import {
  PMM_NEW_NAV_PATH,
  SEP_ATW_PATH,
  SEP_MYSQL_BACKUPS_PATH,
  POM_PATH,
  LEGACY_SEP_POM_PATH,
} from 'lib/constants';
import { RealtimeSessionsPage } from 'pages/rta/sessions';
import { Redirect, SettingsRedirect } from 'components/redirect';
import RealtimeOverviewPage from 'pages/rta/overview/RealtimeOverview';
import RealtimeTab from 'pages/rta/tab/RealtimeTab';
import { AlertsPage } from 'pages/alerting/status';
import { AtwApp } from '@sep/plugins-atw';
import { PomApp } from '@sep/plugins-pom';
import { PomPage } from 'pom/PomPage';
import { SchemaDrivenPlugin } from '@sep/framework';
import { SepPage } from './sep/SepPage';

// Route paths below are relative to the `PMM_NEW_NAV_PATH` parent, while the
// shared SEP constants are absolute (the nav and each plugin's `routeBase` need
// them that way), so drop the parent prefix and its separator.
const relativeToNav = (path: string) => path.slice(PMM_NEW_NAV_PATH.length + 1);

/**
 * Send the old `/sep/pom` links to `/pom`, subpath and query intact.
 *
 * `<Navigate to={POM_PATH}>` would drop everything after the mount, landing a
 * bookmarked cluster detail on the overview instead.
 */
const PomRedirect = () => {
  const { pathname, search, hash } = useLocation();
  const moved = pathname.replace(LEGACY_SEP_POM_PATH, POM_PATH);
  return <Navigate to={`${moved}${search}${hash}`} replace />;
};

const router = createBrowserRouter(
  [
    {
      path: '',
      element: <Providers />,
      children: [
        {
          path: PMM_NEW_NAV_PATH,
          element: <MainWithNav />,
          children: [
            {
              path: '',
              element: <Navigate to="graph" />,
            },
            {
              path: 'updates',
              element: <Updates />,
            },
            {
              path: 'updates/clients',
              element: <UpdateClients />,
            },
            {
              path: 'help',
              element: <HelpCenter />,
            },
            {
              path: 'alerting',
              children: [
                {
                  path: 'status',
                  element: <AlertsPage />,
                },
              ],
            },
            {
              path: 'settings/:tab?',
              element: <Settings />,
            },
            {
              path: 'rta',
              children: [
                {
                  path: '',
                  element: <RealtimeTab />,
                },
                {
                  path: 'selection',
                  element: <RealtimeSelection />,
                },
                {
                  path: 'sessions',
                  element: <RealtimeSessionsPage />,
                },
                {
                  path: 'overview',
                  element: <RealtimeOverviewPage />,
                },
              ],
            },
            // SEP apps mounted as native routes. Both plugins compose their own
            // <Routes>, so the paths are splats.
            {
              // ATW ("Collect Diagnostic Data") is now incident-first: AtwApp
              // composes its own <Routes> (incident list at index, workspace at
              // :incidentId), so this must be a splat. Backend API calls hit
              // /apps/atw and the incident / batch-execution endpoints.
              path: `${relativeToNav(SEP_ATW_PATH)}/*`,
              element: (
                <SepPage>
                  <AtwApp />
                </SepPage>
              ),
            },
            {
              // POM (PSMDB Open Manager) is bespoke like ATW: PomApp composes its
              // own <Routes>, so this must be a splat.
              //
              // All four pages are under it now. Discovery used to be mounted
              // separately inside SepPage, because it read SEP's app directly and
              // needed a bearer minted from the PMM session; pmm-managed proxies the
              // estate at /v1/pom/inventory, so no POM path in the browser talks to
              // SEP any more. That is why PomPage rather than SepPage, and why a sick
              // SEP now shows an error inside a page that still renders instead of
              // blanking it -- SepAuthGate fails closed.
              path: `${relativeToNav(POM_PATH)}/*`,
              element: (
                <PomPage>
                  <PomApp />
                </PomPage>
              ),
            },
            {
              // POM answered under /sep for as long as its backend was a SEP app.
              // Kept so links and bookmarks from that era still land.
              path: `${relativeToNav(LEGACY_SEP_POM_PATH)}/*`,
              element: <PomRedirect />,
            },
            {
              path: `${relativeToNav(SEP_MYSQL_BACKUPS_PATH)}/*`,
              // routeBase must match the mount path (basename-stripped) so the
              // plugin's absolute nav — detail back/edit/schedule links and the
              // related-app (Restore) tab bar — resolves under /sep, not the
              // SEP-default /apps/{name}. PMM_NEW_NAV_PATH is '' so this is the
              // full path below the /pmm-ui basename.
              element: (
                <SepPage>
                  <SchemaDrivenPlugin
                    pluginName="mysql_backups"
                    routeBase={SEP_MYSQL_BACKUPS_PATH}
                  />
                </SepPage>
              ),
            },
            // Fallback
            {
              path: 'graph/settings/:tab?',
              element: <SettingsRedirect />,
            },
            {
              path: 'graph/alerting/alerts',
              element: <Navigate to="/alerting/status" replace />,
            },
            // Grafana routes are handled at the Main component level
            {
              path: 'graph/*',
              element: <React.Fragment />,
            },
            {
              path: '*',
              element: <NotFoundPage />,
            },
          ],
        },
        // Provide fallback for /next/* paths to redirect to the root path
        {
          path: '/next/*',
          element: <Redirect />,
        },
        {
          path: '*',
          element: <div>Not found!</div>,
        },
      ],
    },
  ],
  {
    basename: '/pmm-ui',
  }
);

export default router;
