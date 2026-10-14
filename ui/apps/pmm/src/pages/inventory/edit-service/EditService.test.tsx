import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { changeRdsCredentials, getRdsExporter } from 'api/agents';
import { changeService, listManagedServices } from 'api/services';
import { Route, Routes, useLocation } from 'react-router-dom';
import type { RdsExporter } from 'types/agent.types';
import { TEST_MANAGED_SERVICE } from 'utils/testStubs';
import {
  wrapWithQueryProvider,
  wrapWithSnackbarProvider,
} from 'utils/testUtils';
import { TestWrapper } from 'utils/testWrapper';
import { EditService } from './EditService';

vi.mock('api/services');
vi.mock('api/agents', () => ({
  getAgentVersions: vi.fn(),
  listAgents: vi.fn(),
  removeAgent: vi.fn(),
  getRdsExporter: vi.fn(),
  changeRdsCredentials: vi.fn(),
}));

const SERVICE = {
  ...TEST_MANAGED_SERVICE,
  serviceId: 'service-1',
  serviceName: 'mysql-1',
  nodeId: 'node-1',
  environment: 'prod',
  cluster: 'cluster-1',
  replicationSet: 'rs-1',
  customLabels: { team: 'db' },
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

const saveAndConfirm = async () => {
  fireEvent.click(screen.getByTestId('edit-service-submit'));
  const modal = await screen.findByTestId('confirm-changes-modal');
  fireEvent.click(within(modal).getByTestId('confirm-changes-modal-confirm'));

  return modal;
};

describe('EditService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(listManagedServices).mockResolvedValue({ services: [SERVICE] });
    vi.mocked(getRdsExporter).mockResolvedValue(null);
    vi.mocked(changeService).mockResolvedValue();
    vi.mocked(changeRdsCredentials).mockResolvedValue();
  });

  it('prefills the current labels', async () => {
    renderEditService();

    expect(
      await screen.findByRole('heading', { name: 'Editing “mysql-1” service' })
    ).toBeInTheDocument();
    await screen.findByTestId('edit-service-form');
    expect(field('Environment').value).toBe('prod');
    expect(field('Cluster').value).toBe('cluster-1');
    expect(field('Replication set').value).toBe('rs-1');
    expect(field('Custom labels').value).toBe('team:db');
    expect(getRdsExporter).toHaveBeenCalledWith('node-1');
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
    await waitFor(() =>
      expect(changeService).toHaveBeenCalledWith({
        serviceId: 'service-1',
        cluster: 'cluster-2',
      })
    );
    expect(changeRdsCredentials).not.toHaveBeenCalled();
    expect(await screen.findByTestId('location-probe')).toHaveTextContent(
      '/inventory/services'
    );
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
    expect(changeService).not.toHaveBeenCalled();
    expect(await screen.findByTestId('location-probe')).toBeInTheDocument();
  });

  it('leaves the labels alone when the credentials are rejected', async () => {
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
    expect(changeService).not.toHaveBeenCalled();
    expect(screen.queryByTestId('location-probe')).not.toBeInTheDocument();
  });

  it('saves an unrelated change next to a label value with colons', async () => {
    vi.mocked(listManagedServices).mockResolvedValue({
      services: [{ ...SERVICE, customLabels: { endpoint: 'db:3306' } }],
    });
    renderEditService();

    await screen.findByTestId('edit-service-form');
    expect(field('Custom labels').value).toBe('endpoint:db:3306');
    changeField('Environment', 'staging');
    await saveAndConfirm();

    await waitFor(() =>
      expect(changeService).toHaveBeenCalledWith({
        serviceId: 'service-1',
        environment: 'staging',
      })
    );
  });

  it('saves an unrelated change next to a label value with spaces', async () => {
    vi.mocked(listManagedServices).mockResolvedValue({
      services: [{ ...SERVICE, customLabels: { owner: 'DB Team:Lead' } }],
    });
    renderEditService();

    await screen.findByTestId('edit-service-form');
    expect(field('Custom labels').value).toBe('owner:DB Team:Lead');
    changeField('Environment', 'staging');
    await saveAndConfirm();

    await waitFor(() =>
      expect(changeService).toHaveBeenCalledWith({
        serviceId: 'service-1',
        environment: 'staging',
      })
    );
  });

  it('does not report a failed request as a missing service', async () => {
    vi.mocked(listManagedServices).mockRejectedValue(new Error('timeout'));
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
    renderEditService('missing');

    expect(
      await screen.findByText('This service no longer exists.')
    ).toBeInTheDocument();
  });
});
