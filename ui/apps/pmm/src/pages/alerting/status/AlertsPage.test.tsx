import { fireEvent, render, screen, within } from '@testing-library/react';
import { useLocation, useNavigate } from 'react-router-dom';
import { TestWrapper } from 'utils/testWrapper';
import {
  wrapWithQueryProvider,
  wrapWithSettings,
  wrapWithSnackbarProvider,
} from 'utils/testUtils';
import type { PrometheusAlertRulesResponse } from 'types/alerting.types';
import AlertsPage from './AlertsPage';

const RULES: PrometheusAlertRulesResponse = {
  data: {
    groups: [
      {
        uid: 'group-1',
        name: 'Group',
        rules: [
          {
            uid: 'rule-cpu',
            name: 'High CPU',
            state: 'firing',
            alerts: ['node-1', 'node-2'].map((node) => ({
              labels: { alertname: 'High CPU', node_name: node },
              annotations: {},
              state: 'Alerting' as const,
            })),
          },
        ],
      },
    ],
  },
};

const NODE_1_LINK =
  'rule=rule-cpu&matcher=alertname%3DHigh+CPU&matcher=node_name%3Dnode-1';
const NODE_2_LINK =
  'rule=rule-cpu&matcher=alertname%3DHigh+CPU&matcher=node_name%3Dnode-2';

vi.mock('hooks/api/usePrometheusAlertRules', () => ({
  usePrometheusAlertRules: () => ({
    data: RULES,
    isLoading: false,
    isError: false,
  }),
}));
// Stable references: a fresh [] per render would rebuild the rows every render.
const { NO_SILENCE_DATA } = vi.hoisted(() => ({
  NO_SILENCE_DATA: { data: [] },
}));

vi.mock('hooks/api/useAlertmanagerAlerts', () => ({
  useAlertmanagerAlerts: () => NO_SILENCE_DATA,
}));
vi.mock('hooks/api/useAlertmanagerSilences', () => ({
  useAlertmanagerSilences: () => NO_SILENCE_DATA,
}));
vi.mock('hooks/api/useGrafanaRuler', () => ({
  useGrafanaRulerRule: () => ({ data: undefined, isLoading: false }),
}));

const LocationDisplay = () => {
  const location = useLocation();
  const navigate = useNavigate();

  return (
    <>
      <div data-testid="location">{location.pathname + location.search}</div>
      <button type="button" onClick={() => navigate(-1)}>
        Browser back
      </button>
    </>
  );
};

const renderPage = (url = '/alerting/status') =>
  render(
    wrapWithQueryProvider(
      wrapWithSnackbarProvider(
        wrapWithSettings(
          <TestWrapper routerProps={{ initialEntries: [url] }}>
            <AlertsPage />
            <LocationDisplay />
          </TestWrapper>,
          { settings: { alertingEnabled: true } }
        )
      )
    )
  );

const location = () => screen.getByTestId('location').textContent;
const pane = () => screen.getByTestId('alert-details-pane');

const openFromTable = async (rowIndex: number) => {
  fireEvent.click(
    screen.getAllByRole('button', { name: /row actions/i })[rowIndex]
  );
  fireEvent.click(await screen.findByText('Notification details'));
};

describe('AlertsPage alert links', () => {
  it('opens the alert the URL links to', async () => {
    renderPage(`/alerting/status?${NODE_2_LINK}`);

    expect(await within(pane()).findByText('node-2')).toBeInTheDocument();
    expect(pane()).toHaveAttribute('aria-hidden', 'false');
  });

  it('opens the tab the URL names', async () => {
    renderPage(`/alerting/status?${NODE_2_LINK}&tab=raw-data`);

    expect(
      await screen.findByTestId('alert-details-pane-raw-data-tab')
    ).toHaveAttribute('aria-selected', 'true');
  });

  it('writes the alert opened from the table to the URL', async () => {
    renderPage();

    await openFromTable(0);

    expect(location()).toBe(`/alerting/status?${NODE_1_LINK}`);
  });

  it('writes the selected tab to the URL', async () => {
    renderPage(`/alerting/status?${NODE_1_LINK}`);

    fireEvent.click(
      await screen.findByTestId('alert-details-pane-raw-data-tab')
    );

    expect(location()).toBe(`/alerting/status?${NODE_1_LINK}&tab=raw-data`);
  });

  it('removes the link but keeps other params when the pane closes', async () => {
    renderPage(`/alerting/status?kiosk=true&${NODE_1_LINK}&tab=raw-data`);

    fireEvent.click(
      await screen.findByTestId('alert-details-pane-close-button')
    );

    expect(location()).toBe('/alerting/status?kiosk=true');
    expect(pane()).toHaveAttribute('aria-hidden', 'true');
  });

  it('closes the pane on Back, without stepping through next/previous', async () => {
    renderPage();

    await openFromTable(0);
    fireEvent.click(screen.getByTestId('alert-details-pane-next-button'));
    expect(location()).toBe(`/alerting/status?${NODE_2_LINK}`);

    fireEvent.click(screen.getByText('Browser back'));

    expect(location()).toBe('/alerting/status');
    expect(pane()).toHaveAttribute('aria-hidden', 'true');
  });

  it('keeps the tab, last in the URL, when moving to the next alert', async () => {
    renderPage(`/alerting/status?${NODE_1_LINK}&tab=raw-data`);

    fireEvent.click(
      await screen.findByTestId('alert-details-pane-next-button')
    );

    expect(location()).toBe(`/alerting/status?${NODE_2_LINK}&tab=raw-data`);
  });

  it('shows a notice and drops the link when the alert no longer exists', async () => {
    renderPage(
      '/alerting/status?rule=rule-cpu&matcher=node_name%3Dremoved-node'
    );

    expect(
      await screen.findByText(/this alert is no longer active/i)
    ).toBeInTheDocument();
    expect(location()).toBe('/alerting/status');
    expect(pane()).toHaveAttribute('aria-hidden', 'true');
  });

  it('hides the notice once another alert is opened', async () => {
    renderPage(
      '/alerting/status?rule=rule-cpu&matcher=node_name%3Dremoved-node'
    );
    await screen.findByText(/this alert is no longer active/i);

    await openFromTable(0);

    expect(
      screen.queryByText(/this alert is no longer active/i)
    ).not.toBeInTheDocument();
  });

  it('copies the link to the open alert', async () => {
    renderPage(`/alerting/status?${NODE_2_LINK}&tab=raw-data`);

    fireEvent.click(
      await screen.findByTestId('alert-details-pane-copy-link-button')
    );

    expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
      `${window.location.origin}/alerting/status?${NODE_2_LINK}&tab=raw-data`
    );
    expect(
      await screen.findByText('Link copied to clipboard')
    ).toBeInTheDocument();
  });

  it('says so when the clipboard is not available', async () => {
    vi.stubGlobal('isSecureContext', false);
    renderPage(`/alerting/status?${NODE_2_LINK}`);

    fireEvent.click(
      await screen.findByTestId('alert-details-pane-copy-link-button')
    );

    expect(
      await screen.findByText('Clipboard is not available')
    ).toBeInTheDocument();
    expect(navigator.clipboard.writeText).not.toHaveBeenCalled();
    vi.unstubAllGlobals();
  });
});
