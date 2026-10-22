import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { listManagedServices, removeService } from 'api/services';
import { Route, Routes } from 'react-router-dom';
import { AgentStatus, AgentType } from 'types/agent.types';
import { ServiceStatus, type ManagedService } from 'types/services.types';
import { TEST_MANAGED_SERVICE } from 'utils/testStubs';
import {
  wrapWithQueryProvider,
  wrapWithSnackbarProvider,
} from 'utils/testUtils';
import { TestWrapper } from 'utils/testWrapper';
import { CLUSTERS_SWITCH_KEY } from '../Inventory.constants';
import { Services } from './Services';
import { Messages } from './Services.messages';

vi.mock('api/services');

const service = (
  serviceId: string,
  serviceName: string,
  cluster: string
): ManagedService => ({
  ...TEST_MANAGED_SERVICE,
  serviceId,
  serviceName,
  cluster,
  nodeId: 'node-1',
  nodeName: 'node-1',
  status: ServiceStatus.up,
  agents: [
    {
      agentId: `${serviceId}-exporter`,
      agentType: AgentType.mysqldExporter,
      status: AgentStatus.running,
      isConnected: false,
      disabled: false,
      properties: {},
    },
  ],
  labels: { environment: 'prod' },
});

const SERVICES = [
  service('service-1', 'mysql-1', 'cluster-1'),
  service('service-2', 'mysql-2', 'cluster-1'),
  service('service-3', 'pg-1', ''),
];

const renderServices = (path = '/inventory/services') =>
  render(
    <TestWrapper routerProps={{ initialEntries: [path] }}>
      {wrapWithQueryProvider(
        wrapWithSnackbarProvider(
          <Routes>
            <Route path="/inventory/services" element={<Services />} />
          </Routes>
        )
      )}
    </TestWrapper>
  );

const confirmDeletion = async () => {
  const modal = await screen.findByTestId('inventory-delete-modal');
  fireEvent.click(within(modal).getByTestId('inventory-delete-modal-confirm'));
};

describe('Services', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    vi.mocked(listManagedServices).mockResolvedValue({ services: SERVICES });
    vi.mocked(removeService).mockResolvedValue();
  });

  it('lists the services, linking to their node', async () => {
    renderServices();

    expect(await screen.findByText('mysql-1')).toBeInTheDocument();
    expect(screen.getByText('pg-1')).toBeInTheDocument();
    expect(screen.getAllByRole('link', { name: 'node-1' })[0]).toHaveAttribute(
      'href',
      '/inventory/nodes?nodes.f.nodeId=node-1'
    );
    expect(screen.getByTestId('add-service-button')).toHaveAttribute(
      'href',
      '/graph/add-instance'
    );
  });

  it('explains the address column in its header', async () => {
    renderServices();

    await screen.findByText('mysql-1');
    const header = screen.getByRole('columnheader', { name: /Address/ });
    fireEvent.mouseOver(within(header).getByTestId('address-info-icon'));

    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      Messages.addressInfo
    );
  });

  it('deletes a service from its actions, in force mode', async () => {
    renderServices();

    await screen.findByText('mysql-1');
    fireEvent.click(screen.getAllByRole('button', { name: 'Row Actions' })[0]);
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Delete' }));

    const modal = await screen.findByTestId('inventory-delete-modal');
    expect(
      within(modal).getByTestId('inventory-delete-modal-message')
    ).toHaveTextContent('By deleting service “mysql-1”');
    fireEvent.click(within(modal).getByRole('checkbox'));
    await confirmDeletion();

    await waitFor(() =>
      expect(removeService).toHaveBeenCalledWith({
        serviceId: 'service-1',
        force: true,
      })
    );
  });

  it('deletes the selected services', async () => {
    renderServices();

    await screen.findByText('mysql-1');
    expect(screen.getByTestId('delete-services-button')).toBeDisabled();

    const [firstRow, secondRow] = screen.getAllByRole('checkbox', {
      name: 'Toggle select row',
    });
    fireEvent.click(firstRow);
    fireEvent.click(secondRow);
    fireEvent.click(screen.getByTestId('delete-services-button'));

    expect(
      await screen.findByTestId('inventory-delete-modal-message')
    ).toHaveTextContent('permanently delete 2 services');
    await confirmDeletion();

    await waitFor(() => expect(removeService).toHaveBeenCalledTimes(2));
    expect(removeService).toHaveBeenCalledWith({
      serviceId: 'service-1',
      force: false,
    });
    expect(removeService).toHaveBeenCalledWith({
      serviceId: 'service-2',
      force: false,
    });
  });

  it('organizes the services by clusters', async () => {
    renderServices();

    await screen.findByText('mysql-1');
    fireEvent.click(
      screen.getByRole('switch', { name: 'Organize by Clusters' })
    );

    expect(await screen.findByText('cluster-1')).toBeInTheDocument();
    expect(screen.getByText('Non-clustered services')).toBeInTheDocument();
    expect(screen.queryByText('mysql-1')).not.toBeInTheDocument();
    expect(localStorage.getItem(CLUSTERS_SWITCH_KEY)).toBe('true');
  });

  it('filters on a Warning monitoring status', async () => {
    const warning = service('service-4', 'mysql-3', '');
    vi.mocked(listManagedServices).mockResolvedValue({
      services: [
        ...SERVICES,
        { ...warning, agents: [{ ...warning.agents[0], disabled: true }] },
      ],
    });
    renderServices('/inventory/services?services.f.agentsStatus=Warning');

    expect(await screen.findByText('mysql-3')).toBeInTheDocument();
    expect(screen.queryByText('mysql-1')).not.toBeInTheDocument();
    expect(
      within(
        screen.getByRole('columnheader', { name: /Monitoring/ })
      ).getByText('Warning')
    ).toBeInTheDocument();
  });

  it('filters on the service from the URL, showing its ID column', async () => {
    renderServices('/inventory/services?services.f.serviceId=service-2');

    expect(await screen.findByText('mysql-2')).toBeInTheDocument();
    expect(screen.queryByText('mysql-1')).not.toBeInTheDocument();
    expect(screen.queryByText('pg-1')).not.toBeInTheDocument();
    expect(
      screen.getByRole('columnheader', { name: /Service ID/ })
    ).toBeInTheDocument();
  });

  it('shows the service ID column from the columns menu', async () => {
    renderServices();

    await screen.findByText('mysql-1');
    expect(
      screen.queryByRole('columnheader', { name: /Service ID/ })
    ).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Show/Hide columns' }));
    const menu = await screen.findByRole('menu');
    fireEvent.click(within(menu).getByText('Service ID'));
    fireEvent.keyDown(menu, { key: 'Escape' });

    expect(
      await screen.findByRole('columnheader', { name: /Service ID/ })
    ).toBeInTheDocument();
    expect(
      JSON.parse(localStorage.getItem('inventory-services.columns') ?? '{}')
    ).toMatchObject({ serviceId: true });
  });

  it('keeps the service ID column hidden after a change made while filtered', async () => {
    renderServices('/inventory/services?services.f.serviceId=service-2');

    await screen.findByText('mysql-2');
    fireEvent.click(screen.getByRole('button', { name: 'Show/Hide columns' }));
    const menu = await screen.findByRole('menu');
    fireEvent.click(within(menu).getByText('Port'));
    fireEvent.keyDown(menu, { key: 'Escape' });

    await waitFor(() =>
      expect(
        JSON.parse(localStorage.getItem('inventory-services.columns') ?? '{}')
      ).toMatchObject({ port: false, serviceId: false })
    );
  });

  it('expands the clusters holding the filtered services', async () => {
    localStorage.setItem(CLUSTERS_SWITCH_KEY, 'true');
    renderServices('/inventory/services?services.f.serviceId=service-2');

    expect(await screen.findByText('mysql-2')).toBeInTheDocument();
    expect(screen.getByText('cluster-1')).toBeInTheDocument();
    expect(
      screen.queryByText('Non-clustered services')
    ).not.toBeInTheDocument();
  });
});
