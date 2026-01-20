import React from 'react';
import { Navigate, createBrowserRouter } from 'react-router-dom';
import { Updates } from 'pages/updates';
import { UpdateClients } from 'pages/update-clients/UpdateClients';
import { MainWithNav } from 'components/main/MainWithNav';
import { NotFoundPage } from 'pages/not-found';
import { HelpCenter } from 'pages/help-center';
import { RealTimeSelection } from 'pages/rta/selection';
import Providers from 'Providers';
import { PMM_NEW_NAV_PATH } from 'lib/constants';
import { Redirect } from 'components/redirect';

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
              path: 'rta',
              element: <RealTimeSelection />,
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
