import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { changeRdsCredentials, getRdsExporter } from 'api/agents';
import { toServiceAgent } from 'api/inventory.utils';
import { getNode } from 'api/nodes';
import { getEditableService, updateService } from 'api/services';
import { Route, Routes, useLocation } from 'react-router-dom';
import {
  AgentType,
  type RdsExporter,
  type ServiceAgent,
} from 'types/agent.types';
import { type ManagedNode, NodeType } from 'types/nodes.types';
import {
  type EditableService,
  type InventoryService,
  ManagedServiceType,
  ServiceStatus,
} from 'types/services.types';
import {
  wrapWithQueryProvider,
  wrapWithSnackbarProvider,
} from 'utils/testUtils';
import { TestWrapper } from 'utils/testWrapper';
import { EditService } from './EditService';

vi.mock('api/services');
vi.mock('api/nodes');
vi.mock('api/agents', () => ({
  getAgentVersions: vi.fn(),
  listAgents: vi.fn(),
  removeAgent: vi.fn(),
  getRdsExporter: vi.fn(),
  changeRdsCredentials: vi.fn(),
}));

const SERVICE: InventoryService = {
  serviceId: 'service-1',
  serviceType: ManagedServiceType.mysql,
  serviceName: 'mysql-1',
  nodeId: 'node-1',
  address: '10.0.0.5',
  port: 3306,
  socket: '',
  databaseName: '',
  group: '',
  environment: 'prod',
  cluster: 'cluster-1',
  replicationSet: 'rs-1',
  customLabels: { team: 'db' },
};

const agent = (
  agentType: AgentType,
  overrides: Partial<ServiceAgent> = {}
): ServiceAgent => ({
  ...toServiceAgent({ agent_id: `${agentType}-1`, agent_type: agentType }),
  ...overrides,
});

const MYSQL_AGENTS = [
  agent(AgentType.mysqldExporter, { username: 'pmm', isPasswordSet: true }),
  agent(AgentType.qanMysqlPerfschemaAgent),
];

const editable = (
  overrides: Partial<InventoryService> = {},
  agents: ServiceAgent[] = MYSQL_AGENTS
): EditableService => ({ service: { ...SERVICE, ...overrides }, agents });

const NODE: ManagedNode = {
  nodeId: 'node-1',
  nodeType: NodeType.generic,
  nodeName: 'db-host-1',
  address: '10.0.0.5',
  status: ServiceStatus.up,
  customLabels: {},
  isPmmServerNode: false,
  agents: [],
  services: [],
  properties: {},
};

const KEY_EXPORTER: RdsExporter = {
  agentId: 'rds-1',
  awsAccessKey: 'AKIA',
  awsRoleArn: '',
  isAwsSecretKeySet: true,
};

const ROLE_ARN = 'arn:aws:iam::123456789012:role/PmmRdsMonitoring';

const LocationProbe = () => (
  <span data-testid="location-probe">{useLocation().pathname}</span>
);

const renderEditService = (serviceId = 'service-1') =>
  render(
    <TestWrapper
      routerProps={{
        initialEntries: [`/inventory/services/${serviceId}/edit`],
      }}
    >
      {wrapWithQueryProvider(
        wrapWithSnackbarProvider(
          <Routes>
            <Route
              path="/inventory/services/:serviceId/edit"
              element={<EditService />}
            />
            <Route path="/inventory/services" element={<LocationProbe />} />
          </Routes>
        )
      )}
    </TestWrapper>
  );

const field = (name: string) =>
  screen.getByRole('textbox', { name }) as HTMLInputElement;

const changeField = (name: string, value: string) =>
  fireEvent.change(field(name), { target: { value } });

const changeInput = (label: string, value: string) =>
  fireEvent.change(screen.getByLabelText(label), { target: { value } });

const toggle = (name: string) =>
  fireEvent.click(screen.getByRole('switch', { name }));

const saveAndConfirm = async () => {
  fireEvent.click(screen.getByTestId('edit-service-submit'));
  const modal = await screen.findByTestId('confirm-changes-modal');
  fireEvent.click(within(modal).getByTestId('confirm-changes-modal-confirm'));

  return modal;
};

describe('EditService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(getEditableService).mockResolvedValue(editable());
    vi.mocked(getNode).mockResolvedValue(NODE);
    vi.mocked(getRdsExporter).mockResolvedValue(null);
    vi.mocked(updateService).mockResolvedValue('');
    vi.mocked(changeRdsCredentials).mockResolvedValue();
  });

  it('prefills the current labels and settings', async () => {
    renderEditService();

    expect(
      await screen.findByRole('heading', { name: 'Editing “mysql-1” service' })
    ).toBeInTheDocument();
    await screen.findByTestId('edit-service-form');
    expect(field('Environment').value).toBe('prod');
    expect(field('Cluster').value).toBe('cluster-1');
    expect(field('Replication set').value).toBe('rs-1');
    expect(field('Custom labels').value).toBe('team:db');
    expect(field('Username').value).toBe('pmm');
    expect(screen.getByLabelText('Password')).toHaveValue('');
    expect(
      screen.getByRole('radio', { name: 'Performance Schema' })
    ).toBeChecked();
    expect(screen.getByRole('radio', { name: 'Pull' })).toBeChecked();
    expect(getEditableService).toHaveBeenCalledWith('service-1');
    expect(getRdsExporter).toHaveBeenCalledWith('node-1');
  });

  it('shows what cannot be changed as read-only details', async () => {
    renderEditService();

    const details = await screen.findByTestId('service-details');
    expect(within(details).getByText('MySQL')).toBeInTheDocument();
    expect(within(details).getByText('db-host-1')).toBeInTheDocument();
    expect(within(details).getByText('10.0.0.5:3306')).toBeInTheDocument();
  });

  it('leaves the credentials section out without an RDS exporter', async () => {
    renderEditService();

    await screen.findByTestId('edit-service-form');
    expect(screen.queryByTestId('rds-credentials')).not.toBeInTheDocument();
  });

  it('saves only the changed labels, warning about the cluster', async () => {
    renderEditService();

    await screen.findByTestId('edit-service-form');
    changeField('Cluster', 'cluster-2');
    const modal = await saveAndConfirm();

    expect(
      within(modal).getByTestId('cluster-changed-warning')
    ).toBeInTheDocument();
    expect(
      within(modal).queryByTestId('settings-changed-info')
    ).not.toBeInTheDocument();
    await waitFor(() =>
      expect(updateService).toHaveBeenCalledWith({
        serviceId: 'service-1',
        serviceType: ManagedServiceType.mysql,
        changes: { cluster: 'cluster-2' },
      })
    );
    expect(changeRdsCredentials).not.toHaveBeenCalled();
    expect(await screen.findByTestId('location-probe')).toHaveTextContent(
      '/inventory/services'
    );
  });

  it('sends a typed password and says the agents restart', async () => {
    renderEditService();

    await screen.findByTestId('edit-service-form');
    expect(
      screen.getByText('Leave blank to keep the current one')
    ).toBeInTheDocument();
    changeInput('Password', 'new-secret');
    const modal = await saveAndConfirm();

    expect(
      within(modal).getByTestId('settings-changed-info')
    ).toBeInTheDocument();
    await waitFor(() =>
      expect(updateService).toHaveBeenCalledWith({
        serviceId: 'service-1',
        serviceType: ManagedServiceType.mysql,
        changes: { password: 'new-secret' },
      })
    );
  });

  it('skips the connection check for that save only', async () => {
    vi.mocked(updateService).mockRejectedValueOnce(new Error('rejected'));
    renderEditService();

    await screen.findByTestId('edit-service-form');
    changeInput('Password', 'new-secret');
    fireEvent.click(screen.getByTestId('edit-service-submit'));
    let modal = await screen.findByTestId('confirm-changes-modal');
    fireEvent.click(
      within(modal).getByRole('checkbox', { name: 'Skip the connection check' })
    );
    fireEvent.click(within(modal).getByTestId('confirm-changes-modal-confirm'));

    await waitFor(() =>
      expect(updateService).toHaveBeenCalledWith({
        serviceId: 'service-1',
        serviceType: ManagedServiceType.mysql,
        changes: { password: 'new-secret', skipConnectionCheck: true },
      })
    );
    await waitFor(() =>
      expect(
        screen.queryByTestId('confirm-changes-modal')
      ).not.toBeInTheDocument()
    );

    fireEvent.click(screen.getByTestId('edit-service-submit'));
    modal = await screen.findByTestId('confirm-changes-modal');
    expect(
      within(modal).getByRole('checkbox', { name: 'Skip the connection check' })
    ).not.toBeChecked();
  });

  it('offers to skip the connection check only for settings', async () => {
    renderEditService();

    await screen.findByTestId('edit-service-form');
    changeField('Cluster', 'cluster-2');
    fireEvent.click(screen.getByTestId('edit-service-submit'));
    const modal = await screen.findByTestId('confirm-changes-modal');

    expect(
      within(modal).queryByRole('checkbox', {
        name: 'Skip the connection check',
      })
    ).not.toBeInTheDocument();
  });

  it('switches the query source and shows the server warning', async () => {
    vi.mocked(getEditableService).mockResolvedValue(
      editable({ serviceType: ManagedServiceType.postgresql }, [
        agent(AgentType.postgresExporter, { username: 'pmm' }),
        agent(AgentType.qanPostgresqlPgstatementsAgent),
      ])
    );
    vi.mocked(updateService).mockResolvedValue(
      'Could not to detect the pg_stat_monitor extension on your system. Falling back to the pg_stat_statements.'
    );
    renderEditService();

    await screen.findByTestId('edit-service-form');
    fireEvent.click(screen.getByRole('radio', { name: 'pg_stat_monitor' }));
    await saveAndConfirm();

    await waitFor(() =>
      expect(updateService).toHaveBeenCalledWith({
        serviceId: 'service-1',
        serviceType: ManagedServiceType.postgresql,
        changes: {
          qanPostgresqlPgstatementsAgent: false,
          qanPostgresqlPgstatmonitorAgent: true,
        },
      })
    );
    expect(
      await screen.findByText(/Falling back to the pg_stat_statements/)
    ).toBeInTheDocument();
  });

  // jsdom skips step validation, so this checks what the browser goes by
  it('lets the browser accept a fractional connection timeout', async () => {
    renderEditService();

    await screen.findByTestId('edit-service-form');

    expect(
      screen.getByLabelText('Connection timeout (seconds)')
    ).toHaveAttribute('step', 'any');
  });

  it('hides the QAN settings without a query source', async () => {
    renderEditService();

    await screen.findByTestId('edit-service-form');
    expect(screen.getByLabelText('Max query length')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('radio', { name: 'None' }));

    expect(screen.queryByLabelText('Max query length')).not.toBeInTheDocument();
  });

  it('asks for the certificates once TLS is on', async () => {
    renderEditService();

    await screen.findByTestId('edit-service-form');
    expect(
      screen.queryByLabelText('TLS CA certificate')
    ).not.toBeInTheDocument();
    toggle('Use TLS for database connections');
    changeField('TLS CA certificate', 'CA PEM');
    await saveAndConfirm();

    await waitFor(() =>
      expect(updateService).toHaveBeenCalledWith(
        expect.objectContaining({ changes: { tls: true, tlsCa: 'CA PEM' } })
      )
    );
  });

  it('shows the settings of the service type', async () => {
    vi.mocked(getEditableService).mockResolvedValue(
      editable({ serviceType: ManagedServiceType.mongodb }, [
        agent(AgentType.mongodbExporter, { username: 'pmm' }),
      ])
    );
    renderEditService();

    await screen.findByTestId('edit-service-form');
    expect(
      screen.getByRole('switch', { name: 'Enable all collectors' })
    ).toBeInTheDocument();
    expect(screen.getByLabelText('Max collections limit')).toBeInTheDocument();
    expect(screen.getByRole('radio', { name: 'Profiler' })).toBeInTheDocument();
    expect(
      screen.queryByLabelText('Extra DSN parameters')
    ).not.toBeInTheDocument();
  });

  it('shows the metrics endpoint of an external service', async () => {
    vi.mocked(getEditableService).mockResolvedValue(
      editable({ serviceType: ManagedServiceType.external, address: '' }, [
        agent(AgentType.externalExporter, { listenPort: 9121 }),
      ])
    );
    renderEditService();

    await screen.findByTestId('edit-service-form');
    expect(screen.getByLabelText('Listen port')).toHaveValue(9121);
    expect(field('Group')).toBeInTheDocument();
    expect(screen.queryByTestId('tls-settings')).not.toBeInTheDocument();
    expect(
      screen.queryByTestId('query-analytics-settings')
    ).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Log level')).not.toBeInTheDocument();
  });

  it('offers only the query sources a cloud node allows', async () => {
    vi.mocked(getNode).mockResolvedValue({
      ...NODE,
      nodeType: NodeType.remoteRds,
    });
    renderEditService();

    await screen.findByTestId('edit-service-form');
    await waitFor(() =>
      expect(screen.getByRole('radio', { name: 'Slow log' })).toBeDisabled()
    );
    expect(
      screen.getByRole('radio', { name: 'Performance Schema' })
    ).toBeEnabled();
    expect(screen.getByTestId('query-source-restriction')).toBeInTheDocument();
  });

  it('keeps the query source of the internal PostgreSQL', async () => {
    vi.mocked(getEditableService).mockResolvedValue(
      editable(
        {
          serviceType: ManagedServiceType.postgresql,
          serviceName: 'pmm-server-postgresql',
        },
        [agent(AgentType.postgresExporter, { username: 'pmm' })]
      )
    );
    renderEditService();

    await screen.findByTestId('edit-service-form');
    for (const name of ['None', 'pg_stat_statements', 'pg_stat_monitor']) {
      expect(screen.getByRole('radio', { name })).toBeDisabled();
    }
  });

  it('does not ask for confirmation while the form is invalid', async () => {
    renderEditService();

    await screen.findByTestId('edit-service-form');
    changeField('Custom labels', 'not-a-label');
    fireEvent.click(screen.getByTestId('edit-service-submit'));

    expect(
      await screen.findByText(
        'Values have to be in key:value format, and separated with new line or space'
      )
    ).toBeInTheDocument();
    expect(
      screen.queryByTestId('confirm-changes-modal')
    ).not.toBeInTheDocument();
  });

  it('migrates to an IAM role by clearing both keys in one call', async () => {
    vi.mocked(getRdsExporter).mockResolvedValue(KEY_EXPORTER);
    renderEditService();

    await screen.findByTestId('rds-credentials');
    expect(field('Amazon RDS access key').value).toBe('AKIA');
    fireEvent.click(screen.getByRole('radio', { name: 'IAM role' }));
    changeField('AWS IAM role ARN', ROLE_ARN);
    const modal = await saveAndConfirm();

    expect(
      within(modal).getByTestId('credentials-changed-info')
    ).toBeInTheDocument();
    await waitFor(() =>
      expect(changeRdsCredentials).toHaveBeenCalledWith({
        agentId: 'rds-1',
        credentials: {
          awsRoleArn: ROLE_ARN,
          awsAccessKey: '',
          awsSecretKey: '',
        },
      })
    );
    expect(updateService).not.toHaveBeenCalled();
    expect(await screen.findByTestId('location-probe')).toBeInTheDocument();
  });

  it('leaves the settings alone when the credentials are rejected', async () => {
    vi.mocked(getRdsExporter).mockResolvedValue(KEY_EXPORTER);
    vi.mocked(changeRdsCredentials).mockRejectedValue(new Error('rejected'));
    renderEditService();

    await screen.findByTestId('rds-credentials');
    changeField('Cluster', 'cluster-2');
    fireEvent.click(screen.getByRole('radio', { name: 'Host credentials' }));
    const modal = await saveAndConfirm();

    expect(
      within(modal).getByTestId('host-credentials-warning')
    ).toBeInTheDocument();
    await waitFor(() =>
      expect(
        screen.queryByTestId('confirm-changes-modal')
      ).not.toBeInTheDocument()
    );
    expect(changeRdsCredentials).toHaveBeenCalled();
    expect(updateService).not.toHaveBeenCalled();
    expect(screen.queryByTestId('location-probe')).not.toBeInTheDocument();
  });

  it('reports the settings left unsaved after changed credentials', async () => {
    vi.mocked(getRdsExporter).mockResolvedValue(KEY_EXPORTER);
    vi.mocked(updateService).mockRejectedValue(new Error('rejected'));
    renderEditService();

    await screen.findByTestId('rds-credentials');
    changeField('Cluster', 'cluster-2');
    fireEvent.click(screen.getByRole('radio', { name: 'Host credentials' }));
    await saveAndConfirm();

    expect(
      await screen.findByText(
        'AWS credentials were changed, the other settings were not. Review the error above and save them again.'
      )
    ).toBeInTheDocument();
    expect(screen.queryByTestId('location-probe')).not.toBeInTheDocument();
  });

  it('saves an unrelated change next to a label value with colons', async () => {
    vi.mocked(getEditableService).mockResolvedValue(
      editable({ customLabels: { endpoint: 'db:3306' } })
    );
    renderEditService();

    await screen.findByTestId('edit-service-form');
    expect(field('Custom labels').value).toBe('endpoint:db:3306');
    changeField('Environment', 'staging');
    await saveAndConfirm();

    await waitFor(() =>
      expect(updateService).toHaveBeenCalledWith(
        expect.objectContaining({ changes: { environment: 'staging' } })
      )
    );
  });

  it('saves an unrelated change next to a label value with spaces', async () => {
    vi.mocked(getEditableService).mockResolvedValue(
      editable({ customLabels: { owner: 'DB Team:Lead' } })
    );
    renderEditService();

    await screen.findByTestId('edit-service-form');
    expect(field('Custom labels').value).toBe('owner:DB Team:Lead');
    changeField('Environment', 'staging');
    await saveAndConfirm();

    await waitFor(() =>
      expect(updateService).toHaveBeenCalledWith(
        expect.objectContaining({ changes: { environment: 'staging' } })
      )
    );
  });

  it('does not report a failed request as a missing service', async () => {
    vi.mocked(getEditableService).mockRejectedValue(new Error('timeout'));
    renderEditService();

    await waitFor(() =>
      expect(
        screen.queryByTestId('edit-service-loading')
      ).not.toBeInTheDocument()
    );
    expect(
      screen.queryByText('This service no longer exists.')
    ).not.toBeInTheDocument();
    expect(screen.queryByTestId('edit-service-form')).not.toBeInTheDocument();
  });

  it('says when the service no longer exists', async () => {
    vi.mocked(getEditableService).mockResolvedValue(null);
    renderEditService('missing');

    expect(
      await screen.findByText('This service no longer exists.')
    ).toBeInTheDocument();
  });
});
