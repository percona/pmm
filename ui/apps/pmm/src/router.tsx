import React from 'react';
import { Navigate, createBrowserRouter } from 'react-router-dom';
import { Settings } from 'pages/settings';
import { Updates } from 'pages/updates';
import { UpdateClients } from 'pages/update-clients/UpdateClients';
import { MainWithNav } from 'components/main/MainWithNav';
import { NotFoundPage } from 'pages/not-found';
import { HelpCenter } from 'pages/help-center';
import { RealtimeSelection } from 'pages/rta/selection';
import Providers from 'Providers';
import { PMM_NEW_NAV_PATH } from 'lib/constants';
import { RealtimeSessionsPage } from 'pages/rta/sessions';
import { Redirect, SettingsRedirect } from 'components/redirect';
import RealtimeOverviewPage from 'pages/rta/overview/RealtimeOverview';
import RealtimeTab from 'pages/rta/tab/RealtimeTab';
import { InstallClientPage } from 'pages/install-client';

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
              path: 'install-client',
              element: <InstallClientPage />,
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
            // Fallback
            {
              path: 'graph/settings/:tab?',
              element: <SettingsRedirect />,
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
