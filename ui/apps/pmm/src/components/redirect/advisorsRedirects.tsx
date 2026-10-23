import { Navigate, type RouteObject } from 'react-router-dom';

// Legacy Grafana Advisor paths, served by PMM UI; isMigratedPage keeps them out of the Grafana iframe.
export const advisorsRedirects: RouteObject[] = [
  {
    path: 'graph/advisors/insights/*',
    element: <Navigate to="/advisors/insights" replace />,
  },
  {
    path: 'graph/advisors/*',
    element: <Navigate to="/advisors" replace />,
  },
  {
    path: 'graph/pmm-database-checks/all-checks',
    element: <Navigate to="/advisors" replace />,
  },
  {
    path: 'graph/pmm-database-checks/*',
    element: <Navigate to="/advisors/insights" replace />,
  },
];
