import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { listAgents, removeAgent } from 'api/agents';
import { getNode } from 'api/nodes';
import { getService } from 'api/services';
import { Route, Routes } from 'react-router-dom';
import { AgentStatus, AgentType, type ManagedAgent } from 'types/agent.types';
import { NodeType } from 'types/nodes.types';
import { ServiceStatus } from 'types/services.types';
import { TEST_MANAGED_SERVICE } from 'utils/testStubs';
import {
  wrapWithQueryProvider,
  wrapWithSnackbarProvider,
} from 'utils/testUtils';
import { TestWrapper } from 'utils/testWrapper';
import { Agents } from './Agents';

vi.mock('api/agents', () => ({
  getAgentVersions: vi.fn(),
  listAgents: vi.fn(),
  removeAgent: vi.fn(),
}));
vi.mock('api/nodes');
vi.mock('api/services');

const AGENTS: ManagedAgent[] = [
  {
    agentId: 'agent-1',
    agentType: AgentType.mysqldExporter,
    status: AgentStatus.running,
    isConnected: false,
    disabled: false,
    properties: { listen_port: '42001' },
  },
  {
    agentId: 'agent-2',
    agentType: AgentType.qanMysqlSlowlogAgent,
    status: AgentStatus.waiting,
    isConnected: false,
    disabled: false,
    properties: {},
  },
];

const renderAgents = (path: string) =>
  render(
    <TestWrapper routerProps={{ initialEntries: [path] }}>
      {wrapWithQueryProvider(
        wrapWithSnackbarProvider(
          <Routes>
            <Route
              path="/inventory/services/:serviceId/agents"
              element={<Agents />}
            />
            <Route
              path="/inventory/nodes/:nodeId/agents"
              element={<Agents />}
            />
          </Routes>
        )
      )}
    </TestWrapper>
  );

describe('Agents', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(listAgents).mockResolvedValue(AGENTS);
    vi.mocked(removeAgent).mockResolvedValue();
    vi.mocked(getService).mockResolvedValue({
      ...TEST_MANAGED_SERVICE,
      serviceName: 'mysql-1',
    });
    vi.mocked(getNode).mockResolvedValue({
      nodeId: 'node-1',
      nodeName: 'node-1',
      nodeType: NodeType.generic,
      address: '',
      status: ServiceStatus.up,
      customLabels: {},
      isPmmServerNode: false,
      agents: [],
      services: [],
      properties: {},
    });
  });

  it("lists a service's agents", async () => {
    renderAgents('/inventory/services/service-1/agents');

    expect(await screen.findByTestId('agents-title')).toHaveTextContent(
      'Service mysql-1 / Agents'
    );
    expect(getService).toHaveBeenCalledWith('service-1');
    expect(getNode).not.toHaveBeenCalled();
    expect(listAgents).toHaveBeenCalledWith({ serviceId: 'service-1' });
    expect(screen.getByText('MySQL exporter')).toBeInTheDocument();
    expect(screen.getByText('Running')).toBeInTheDocument();
    expect(screen.getByTestId('agents-go-back')).toHaveAttribute(
      'href',
      '/inventory/services'
    );
  });

  it("lists a node's agents", async () => {
    renderAgents('/inventory/nodes/node-1/agents');

    expect(await screen.findByTestId('agents-title')).toHaveTextContent(
      'Node node-1 / Agents'
    );
    expect(getNode).toHaveBeenCalledWith('node-1');
    expect(getService).not.toHaveBeenCalled();
    expect(listAgents).toHaveBeenCalledWith({ nodeId: 'node-1' });
    expect(screen.getByTestId('agents-go-back')).toHaveAttribute(
      'href',
      '/inventory/nodes'
    );
  });

  it('filters on agents that failed to start', async () => {
    vi.mocked(listAgents).mockResolvedValue([
      ...AGENTS,
      {
        ...AGENTS[0],
        agentId: 'agent-3',
        status: AgentStatus.initializationError,
      },
    ]);
    renderAgents(
      '/inventory/services/service-1/agents?agents.f.status=AGENT_STATUS_INITIALIZATION_ERROR'
    );

    expect(await screen.findByText('agent-3')).toBeInTheDocument();
    expect(screen.queryByText('agent-1')).not.toBeInTheDocument();
    expect(
      within(screen.getByRole('columnheader', { name: /Status/ })).getByText(
        'Initialization error'
      )
    ).toBeInTheDocument();
  });

  it('deletes the selected agents in force mode', async () => {
    renderAgents('/inventory/services/service-1/agents');

    await screen.findByText('agent-2');
    const [firstRow] = screen.getAllByRole('checkbox', {
      name: 'Toggle select row',
    });
    fireEvent.click(firstRow);
    fireEvent.click(screen.getByTestId('delete-agents-button'));

    const modal = await screen.findByTestId('inventory-delete-modal');
    fireEvent.click(within(modal).getByRole('checkbox'));
    fireEvent.click(
      within(modal).getByTestId('inventory-delete-modal-confirm')
    );

    await waitFor(() =>
      expect(removeAgent).toHaveBeenCalledWith({
        agentId: 'agent-1',
        force: true,
      })
    );
    expect(removeAgent).toHaveBeenCalledTimes(1);
  });
});
