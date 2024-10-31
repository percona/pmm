import { Navigate, createBrowserRouter } from 'react-router-dom';
import { Main } from 'components/main/Main';
import { Updates } from 'pages/updates';
import { UpdateClients } from 'pages/update-clients/UpdateClients';
import { DashboardsPage } from 'pages/dashboards';
import { AlertsPage } from 'pages/alerts';

const router = createBrowserRouter(
  [
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
        {
          path: 'd/*',
          element: <DashboardsPage />,
        },
        {
          path: 'alerts',
          element: <AlertsPage />,
        },
      ],
    },
    {
      path: '*',
      element: <Navigate to="/" />,
    },
  ],
  {
    basename: '/pmm-ui',
  }
);

export default router;
