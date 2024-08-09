import { Navigate, createBrowserRouter } from 'react-router-dom';
import { Main } from 'components/main/Main';
import { Updates } from 'pages/updates';
import { UpdateClients } from 'pages/update-clients/UpdateClients';

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
      ],
    },
    {
      path: '*',
      element: <div>Not found!</div>,
    },
  ],
  {
    basename: '/pmm-ui',
  }
);

export default router;
