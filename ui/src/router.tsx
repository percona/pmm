import { Navigate, createBrowserRouter } from 'react-router-dom';
import { Main } from 'components/main/Main';
import { Updates } from 'pages/updates';

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
