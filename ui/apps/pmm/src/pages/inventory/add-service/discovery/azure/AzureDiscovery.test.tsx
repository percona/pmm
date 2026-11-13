import { fireEvent, screen, waitFor } from '@testing-library/react';
import { addAzureService, discoverAzure } from 'api/services';
import {
  DiscoverAzureDatabaseType,
  type DiscoverAzureInstance,
} from 'types/services.types';
import {
  mockNodes,
  renderServiceForm,
  submit,
  type,
  waitForNodePreselect,
} from '../../service-form/ServiceForm.testUtils';
import { AzureDiscovery } from './AzureDiscovery';

vi.mock('api/services', () => ({
  addAzureService: vi.fn(),
  discoverAzure: vi.fn(),
}));
vi.mock('api/nodes', () => ({ listNodes: vi.fn() }));
vi.mock('notistack', () => ({ enqueueSnackbar: vi.fn() }));

const CREDENTIALS = {
  azureClientId: 'client',
  azureClientSecret: 'secret',
  azureTenantId: 'tenant',
  azureSubscriptionId: 'subscription',
};

const MYSQL: DiscoverAzureInstance = {
  instanceId: '/subscriptions/s/resourceGroups/rg/servers/azure-mysql',
  region: 'westeurope',
  serviceName: 'azure-mysql',
  username: 'azureadmin',
  address: 'azure-mysql.mysql.database.azure.com',
  azureResourceGroup: 'rg',
  environment: '',
  type: DiscoverAzureDatabaseType.mysql,
  az: '1',
  nodeModel: 'GP_Gen5_2',
};

const discover = () => {
  type('text-input-azure-client-id', CREDENTIALS.azureClientId);
  type('text-input-azure-client-secret', CREDENTIALS.azureClientSecret);
  type('text-input-azure-tenant-id', CREDENTIALS.azureTenantId);
  type('text-input-azure-subscription-id', CREDENTIALS.azureSubscriptionId);
  submit();
};

describe('AzureDiscovery', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockNodes();
    vi.mocked(discoverAzure).mockResolvedValue({
      azureDatabaseInstance: [MYSQL],
    });
    vi.mocked(addAzureService).mockResolvedValue(undefined);
  });

  it('needs all four credentials to discover', async () => {
    renderServiceForm(AzureDiscovery);

    submit();

    expect(await screen.findAllByText('Required field')).toHaveLength(4);
    expect(discoverAzure).not.toHaveBeenCalled();
  });

  it('adds a discovered MySQL server with the discovery credentials', async () => {
    renderServiceForm(AzureDiscovery);

    discover();
    fireEvent.click(
      await screen.findByTestId(`start-monitoring-${MYSQL.instanceId}`)
    );
    await waitForNodePreselect();
    expect(vi.mocked(discoverAzure).mock.calls[0][0]).toEqual(CREDENTIALS);
    expect(screen.getByTestId('text-input-username')).toHaveValue('azureadmin');
    expect(
      screen.queryByTestId('text-input-disable-collectors')
    ).not.toBeInTheDocument();

    type('text-input-password', 'secret');
    submit();

    await waitFor(() => expect(addAzureService).toHaveBeenCalledTimes(1));
    expect(vi.mocked(addAzureService).mock.calls[0][0]).toEqual(
      expect.objectContaining({
        ...CREDENTIALS,
        type: DiscoverAzureDatabaseType.mysql,
        instanceId: MYSQL.instanceId,
        azureResourceGroup: 'rg',
        serviceName: MYSQL.address,
        nodeName: MYSQL.address,
        port: 3306,
        region: 'westeurope',
        username: 'azureadmin',
        azureDatabaseExporter: true,
        qan: true,
        tablestatsGroupTableLimit: -1,
        pmmAgentId: 'pmm-server-agent',
      })
    );
  });

  it("shows PMM Server's 5s cloud default as the connection timeout", async () => {
    renderServiceForm(AzureDiscovery);
    discover();
    fireEvent.click(
      await screen.findByTestId(`start-monitoring-${MYSQL.instanceId}`)
    );
    await waitForNodePreselect();

    expect(screen.getByTestId('text-input-connection-timeout')).toHaveAttribute(
      'placeholder',
      '5s'
    );
  });

  it('shows the discovery credentials again after cancelling', async () => {
    renderServiceForm(AzureDiscovery);
    discover();
    fireEvent.click(
      await screen.findByTestId(`start-monitoring-${MYSQL.instanceId}`)
    );
    await waitForNodePreselect();

    fireEvent.click(screen.getByTestId('add-service-cancel'));

    expect(await screen.findByTestId('text-input-azure-client-id')).toHaveValue(
      CREDENTIALS.azureClientId
    );
    expect(screen.getByTestId('text-input-azure-subscription-id')).toHaveValue(
      CREDENTIALS.azureSubscriptionId
    );
  });
});
