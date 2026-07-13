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
import AdrePage from 'pages/adre/AdrePage';
import AdreUsagePage from 'pages/adre/AdreUsagePage';
import AdreSettingsPage from 'pages/configuration/AdreSettingsPage';
import AdreDeploymentPage from 'pages/configuration/AdreDeploymentPage';
import InvestigationsListPage from 'pages/investigations/InvestigationsListPage';
import InvestigationDetailPage from 'pages/investigations/InvestigationDetailPage';
import QanPage from 'pages/qan/QanPage';
import QanAiInsightsPage from 'pages/qan/QanAiInsightsPage';

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
              path: 'settings/:tab?',
              element: <Settings />,
            },
            {
              path: 'adre',
              element: <AdrePage />,
            },
            {
              path: 'adre/usage',
              element: <AdreUsagePage />,
            },
            {
              path: 'configuration/ai-assistant',
              element: <AdreSettingsPage />,
            },
            {
              path: 'configuration/ai-deployment',
              element: <AdreDeploymentPage />,
            },
            {
              path: 'investigations',
              element: <InvestigationsListPage />,
            },
            {
              path: 'investigations/:id',
              element: <InvestigationDetailPage />,
            },
            {
              path: 'qan',
              element: <QanPage />,
            },
            {
              path: 'qan/ai-insights',
              element: <QanAiInsightsPage />,
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
