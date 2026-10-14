import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { listNodes, removeNode } from 'api/nodes';
import { Route, Routes } from 'react-router-dom';
import { AgentStatus, AgentType } from 'types/agent.types';
import { type ManagedNode, NodeType } from 'types/nodes.types';
import { ManagedServiceType, ServiceStatus } from 'types/services.types';
import {
  wrapWithQueryProvider,
  wrapWithSnackbarProvider,
} from 'utils/testUtils';
import { TestWrapper } from 'utils/testWrapper';
import { Nodes } from './Nodes';

vi.mock('api/nodes');
vi.mock('api/ha');

const mocks = vi.hoisted(() => ({
  useNodeThresholds: vi.fn(() => ({ data: undefined, isLoading: false })),
}));

vi.mock('hooks/api/useNodeThresholds', () => ({
  useNodeThresholds: mocks.useNodeThresholds,
  useBatchUpdateNodeThresholds: () => ({ mutateAsync: vi.fn() }),
}));
vi.mock('hooks/api/usePrometheusAlertRules', () => ({
  usePrometheusAlertRules: () => ({ data: undefined, isLoading: true }),
}));

const node = (
  nodeId: string,
  nodeName: string,
  overrides: Partial<ManagedNode> = {}
): ManagedNode => ({
  nodeId,
  nodeName,
  nodeType: NodeType.generic,
  address: '10.0.0.1',
  status: ServiceStatus.up,
  customLabels: {},
  isPmmServerNode: false,
  agents: [
    {
      agentId: `${nodeId}-agent`,
      agentType: AgentType.pmmAgent,
      status: AgentStatus.running,
      isConnected: true,
      disabled: false,
      properties: {},
    },
  ],
  services: [],
  properties: {},
  ...overrides,
});

const NODES = [
  node('pmm-server', 'pmm-ha-1', { isPmmServerNode: true }),
  node('node-1', 'node-1', {
    services: [
      {
        serviceId: 'service-1',
        serviceType: ManagedServiceType.mysql,
        serviceName: 'mysql-1',
      },
    ],
  }),
  node('node-2', 'node-2', {
    services: [
      {
        serviceId: 'service-2',
        serviceType: ManagedServiceType.mysql,
        serviceName: 'mysql-2',
      },
      {
        serviceId: 'service-3',
        serviceType: ManagedServiceType.mongodb,
        serviceName: 'mongo-1',
      },
    ],
  }),
];

const renderNodes = (path = '/inventory/nodes') =>
  render(
    <TestWrapper routerProps={{ initialEntries: [path] }}>
      {wrapWithQueryProvider(
        wrapWithSnackbarProvider(
          <Routes>
            <Route path="/inventory/nodes" element={<Nodes />} />
          </Routes>
        )
      )}
    </TestWrapper>
  );

describe('Nodes', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    vi.mocked(listNodes).mockResolvedValue(NODES);
    vi.mocked(removeNode).mockResolvedValue(undefined);
  });

  it('lists the nodes with their HA role and services', async () => {
    renderNodes();

    expect(await screen.findByText('node-1')).toBeInTheDocument();
    expect(await screen.findByTestId('ha-role-chip')).toHaveTextContent(
      'Leader'
    );
    expect(screen.getByRole('link', { name: 'mysql-1' })).toHaveAttribute(
      'href',
      '/inventory/services?services.f.serviceId=service-1'
    );
    expect(screen.getByText('2 services')).toBeInTheDocument();
  });

  it('does not offer to delete the PMM Server node', async () => {
    renderNodes();

    await screen.findByText('pmm-ha-1');
    const serverRow = screen
      .getAllByRole('row')
      .find((row) => within(row).queryByText('pmm-ha-1'));

    expect(
      within(serverRow as HTMLElement).getByRole('checkbox', {
        name: 'Toggle select row',
      })
    ).toBeDisabled();

    fireEvent.click(
      within(serverRow as HTMLElement).getByRole('button', {
        name: 'Row Actions',
      })
    );

    expect(
      await screen.findByRole('menuitem', { name: 'Override alert thresholds' })
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('menuitem', { name: 'Delete' })
    ).not.toBeInTheDocument();
  });

  it("opens a node's alert thresholds", async () => {
    renderNodes();

    await screen.findByText('node-1');
    fireEvent.click(screen.getAllByRole('button', { name: 'Row Actions' })[0]);
    fireEvent.click(
      await screen.findByRole('menuitem', { name: 'Override alert thresholds' })
    );

    expect(
      await screen.findByText('Alert thresholds: node-1')
    ).toBeInTheDocument();
    expect(mocks.useNodeThresholds).toHaveBeenLastCalledWith(
      'node-1',
      expect.objectContaining({ enabled: true })
    );
  });

  it('shows the node ID column only when chosen or filtered on', async () => {
    const { unmount } = renderNodes();

    await screen.findByText('node-1');
    expect(
      screen.queryByRole('columnheader', { name: /Node ID/ })
    ).not.toBeInTheDocument();
    unmount();

    renderNodes('/inventory/nodes?nodes.f.nodeId=node-2');

    expect(
      await screen.findByRole('columnheader', { name: /Node ID/ })
    ).toBeInTheDocument();
    expect(screen.queryByText('node-1')).not.toBeInTheDocument();
  });

  it('drops the node ID filter when its column is hidden', async () => {
    renderNodes('/inventory/nodes?nodes.f.nodeId=node-2');

    await screen.findByRole('columnheader', { name: /Node ID/ });
    expect(screen.queryByText('node-1')).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Show/Hide columns' }));
    const menu = await screen.findByRole('menu');
    fireEvent.click(within(menu).getByText('Node ID'));
    fireEvent.keyDown(menu, { key: 'Escape' });

    expect(await screen.findByText('node-1')).toBeInTheDocument();
    expect(
      screen.queryByRole('columnheader', { name: /Node ID/ })
    ).not.toBeInTheDocument();
  });

  it('lists only the PMM Server nodes', async () => {
    renderNodes();

    await screen.findByText('node-1');
    fireEvent.click(
      screen.getByRole('switch', { name: 'List PMM Server only nodes' })
    );

    await waitFor(() =>
      expect(screen.queryByText('node-1')).not.toBeInTheDocument()
    );
    expect(screen.getByText('pmm-ha-1')).toBeInTheDocument();
  });

  it('keeps the PMM Server only filter from the URL', async () => {
    renderNodes('/inventory/nodes?nodes.f.isPmmServerNode=true');

    expect(await screen.findByText('pmm-ha-1')).toBeInTheDocument();
    expect(screen.queryByText('node-1')).not.toBeInTheDocument();
    expect(
      screen.getByRole('switch', { name: 'List PMM Server only nodes' })
    ).toBeChecked();
  });

  it('deletes a node without force mode unless chosen', async () => {
    renderNodes();

    await screen.findByText('node-1');
    fireEvent.click(screen.getAllByRole('button', { name: 'Row Actions' })[0]);
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Delete' }));

    const modal = await screen.findByTestId('inventory-delete-modal');
    expect(
      within(modal).getByTestId('inventory-delete-modal-message')
    ).toHaveTextContent('permanently delete 1 node?');
    fireEvent.click(
      within(modal).getByTestId('inventory-delete-modal-confirm')
    );

    await waitFor(() =>
      expect(removeNode).toHaveBeenCalledWith({
        nodeId: 'node-1',
        force: false,
      })
    );
  });

  it("shows what a node's removal left behind", async () => {
    vi.mocked(removeNode).mockResolvedValue(
      'Service account of node node-1 still has 2 tokens'
    );
    renderNodes();

    await screen.findByText('node-1');
    fireEvent.click(screen.getAllByRole('button', { name: 'Row Actions' })[0]);
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Delete' }));
    fireEvent.click(
      within(await screen.findByTestId('inventory-delete-modal')).getByTestId(
        'inventory-delete-modal-confirm'
      )
    );

    expect(
      await screen.findByText(
        'Service account of node node-1 still has 2 tokens'
      )
    ).toBeInTheDocument();
    expect(
      screen.getByText('1 of 1 nodes successfully deleted')
    ).toBeInTheDocument();
  });
});
