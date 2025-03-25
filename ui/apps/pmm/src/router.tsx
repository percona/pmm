import { Navigate, createBrowserRouter } from 'react-router-dom';
import { Main } from 'components/main/Main';
import { Updates } from 'pages/updates';
import { UpdateClients } from 'pages/update-clients/UpdateClients';
import { MainWithNav } from 'components/main/MainWithNav';
import { NotFoundPage } from 'pages/not-found';
import { HelpPage } from 'pages/help';
import Providers from 'Providers';

const router = createBrowserRouter(
  [
    {
      path: '',
      element: <Providers />,
      children: [
        {
          path: '/with-nav',
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
              element: <HelpPage />,
            },
            // Grafana routes are handled at the Main component level
            {
              path: 'graph/*',
            },
            {
              path: '*',
              element: <NotFoundPage />,
            },
          ],
        },
        {
          path: '/',
          element: <Main />,
          children: [
            {
              path: '',
              element: <Navigate to="updates" />,
            },
            {
              path: 'updates',
              element: <Updates />,
            },
            {
              path: 'updates/clients',
              element: <UpdateClients />,
            },
          ],
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
