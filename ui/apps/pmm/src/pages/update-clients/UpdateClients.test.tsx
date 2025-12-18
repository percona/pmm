import { fireEvent, render, screen } from '@testing-library/react';
import { UpdateClients } from './UpdateClients';
import { TestWrapper } from 'utils/testWrapper';
import {
  wrapWithQueryProvider,
  wrapWithUpdatesProvider,
} from 'utils/testUtils';
import { AgentUpdateSeverity, GetAgentVersionItem } from 'types/agent.types';
import { Messages } from './UpdateClients.messages';
import * as AgentsApi from 'api/agents';
import * as UpdatesUtils from 'contexts/updates/updates.utils';

vi.mock('api/agents');

const getAgentVersionsMock = vi.spyOn(AgentsApi, 'getAgentVersions');

const getClient = (severity = AgentUpdateSeverity.UP_TO_DATE) => ({
  agentId: `agent-id-${severity.replace('UPDATE_SEVERITY_', '')}`,
  version: '3.0.0',
  nodeName: `node-name-${severity.replace('UPDATE_SEVERITY_', '')}`,
  severity,
});

const renderWithProviders = (clients: GetAgentVersionItem[] = [getClient()]) =>
  render(
    <TestWrapper>
      {wrapWithQueryProvider(
        wrapWithUpdatesProvider(<UpdateClients />, {
          clients,
          areClientsUpToDate: UpdatesUtils.areClientsUpToDate(clients),
        })
      )}
    </TestWrapper>
  );

describe('UpdateClients', () => {
  beforeEach(() => {
    getAgentVersionsMock.mockClear();
  });

  it('shows message when pmm server is up-to-date', () => {
    renderWithProviders();

    expect(
      screen.queryByTestId('pmm-server-up-to-date-alert')
    ).toBeInTheDocument();
  });

  it("doesn't show up-to-date message if clients require updating", () => {
    renderWithProviders([getClient(AgentUpdateSeverity.REQUIRED)]);

    expect(
      screen.queryByTestId('pmm-server-up-to-date-alert')
    ).not.toBeInTheDocument();
  });

  it('shows correct buttons when all clients are up-to-date', () => {
    renderWithProviders();

    expect(
      screen.queryByTestId('how-to-update-clients-link')
    ).not.toBeInTheDocument();
    expect(screen.queryByTestId('refresh-list-button')).toBeInTheDocument();
    expect(screen.queryByTestId('pmm-home-link')).toBeInTheDocument();
  });

  it('shows correct buttons when clients need updating', () => {
    renderWithProviders([getClient(AgentUpdateSeverity.REQUIRED)]);

    expect(
      screen.queryByTestId('how-to-update-clients-link')
    ).toBeInTheDocument();
    expect(screen.queryByTestId('refresh-list-button')).toBeInTheDocument();
    expect(screen.queryByTestId('pmm-home-link')).not.toBeInTheDocument();
  });

  it('shows correctly when no clients are present', () => {
    renderWithProviders([]);

    expect(
      screen.queryByTestId('how-to-update-clients-link')
    ).not.toBeInTheDocument();
    expect(screen.queryByTestId('refresh-list-button')).toBeInTheDocument();
    expect(screen.queryByTestId('pmm-home-link')).toBeInTheDocument();
    expect(screen.queryByText(Messages.table.empty)).toBeInTheDocument();
  });

  it('refreshes data', () => {
    renderWithProviders();

    const refreshButton = screen.getByTestId('refresh-list-button');
    fireEvent.click(refreshButton);

    expect(getAgentVersionsMock).toHaveBeenCalled();
  });

  it('filters only required clients', async () => {
    renderWithProviders([
      getClient(),
      getClient(AgentUpdateSeverity.REQUIRED),
      getClient(AgentUpdateSeverity.CRITICAL),
    ]);

    expect(screen.getAllByRole('row')).toHaveLength(4);

    const select = screen.getByTestId('text-select-button');
    fireEvent.click(select);

    const requiredOption = screen.getByTestId(
      'text-select-option-update-required'
    );
    fireEvent.click(requiredOption);

    expect(screen.getAllByRole('row')).toHaveLength(2);

    expect(screen.getByText('agent-id-REQUIRED')).toBeInTheDocument();
  });

  it('filters only critical clients', async () => {
    renderWithProviders([
      getClient(),
      getClient(AgentUpdateSeverity.REQUIRED),
      getClient(AgentUpdateSeverity.CRITICAL),
    ]);

    expect(screen.getAllByRole('row')).toHaveLength(4);

    const select = screen.getByTestId('text-select-button');
    fireEvent.click(select);

    const criticalOption = screen.getByTestId(
      'text-select-option-critical-to-update'
    );
    fireEvent.click(criticalOption);

    expect(screen.getAllByRole('row')).toHaveLength(2);
    expect(screen.getByText('agent-id-CRITICAL')).toBeInTheDocument();
  });
});
