import { QueryClient } from '@tanstack/react-query';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { AxiosError, type AxiosResponse } from 'axios';
import { discoverRDS } from 'api/services';
import { enqueueSnackbar } from 'notistack';
import {
  DiscoverRDSEngine,
  type DiscoverRDSInstance,
  MetricsMode,
} from 'types/services.types';
import {
  clickLabel,
  mockAddService,
  mockNodes,
  renderServiceForm,
  sentPayload,
  submit,
  type,
  waitForNodePreselect,
} from '../../service-form/ServiceForm.testUtils';
import { RdsDiscovery } from './RdsDiscovery';

vi.mock('api/services', () => ({ addService: vi.fn(), discoverRDS: vi.fn() }));
vi.mock('api/nodes', () => ({ listNodes: vi.fn() }));
vi.mock('notistack', () => ({ enqueueSnackbar: vi.fn() }));

const ROLE_ARN = 'arn:aws:iam::123456789012:role/PmmRdsMonitoring';

const instance = (
  instanceId: string,
  engine: DiscoverRDSEngine
): DiscoverRDSInstance => ({
  region: 'eu-west-1',
  az: 'eu-west-1a',
  instanceId,
  nodeModel: 'db.t3.micro',
  address: `${instanceId}.rds.amazonaws.com`,
  port: engine === DiscoverRDSEngine.mysql ? 3306 : 5432,
  engine,
  engineVersion: '8.0.35',
});

const MYSQL = instance('rds-mysql', DiscoverRDSEngine.mysql);
const POSTGRESQL = instance('rds-pg', DiscoverRDSEngine.postgresql);

// Discovery calls also pass an abort signal or the mutation context.
const lastDiscovery = () => vi.mocked(discoverRDS).mock.calls.at(-1)?.[0];

const discoverWithRole = () => {
  clickLabel('IAM role');
  type('text-input-aws-role-arn', ROLE_ARN);
  submit();
};

const startMonitoring = async (instanceId: string) => {
  fireEvent.click(await screen.findByTestId(`start-monitoring-${instanceId}`));
  await waitForNodePreselect();
};

describe('RdsDiscovery', () => {
  beforeEach(() => {
    mockAddService();
    mockNodes();
    vi.mocked(discoverRDS).mockResolvedValue({
      rdsInstances: [MYSQL, POSTGRESQL],
    });
  });

  it("discovers with PMM Server's own identity on load", async () => {
    renderServiceForm(RdsDiscovery);

    expect(await screen.findByText('rds-mysql')).toBeInTheDocument();
    expect(lastDiscovery()).toEqual({});
    expect(screen.getByText('MySQL 8.0.35')).toBeInTheDocument();
    expect(enqueueSnackbar).not.toHaveBeenCalled();
  });

  it("starts with PMM Server's identity and discovers with it again", async () => {
    renderServiceForm(RdsDiscovery);
    await screen.findByText('rds-mysql');

    expect(screen.getByLabelText("PMM Server's identity")).toBeChecked();
    expect(screen.getByTestId('rds-server-identity-note')).toBeInTheDocument();
    expect(
      screen.queryByTestId('text-input-aws-access-key')
    ).not.toBeInTheDocument();
    submit();

    await waitFor(() => expect(discoverRDS).toHaveBeenCalledTimes(2));
    expect(lastDiscovery()).toEqual({});
  });

  it('discovers with an IAM role ARN', async () => {
    renderServiceForm(RdsDiscovery);
    await screen.findByText('rds-mysql');

    discoverWithRole();

    await waitFor(() => expect(discoverRDS).toHaveBeenCalledTimes(2));
    expect(lastDiscovery()).toEqual({ awsRoleArn: ROLE_ARN });
  });

  it("does not wait for PMM Server's own discovery to discover with a role", async () => {
    vi.mocked(discoverRDS).mockReturnValueOnce(new Promise(() => {}));
    renderServiceForm(RdsDiscovery);

    discoverWithRole();

    expect(await screen.findByText('rds-mysql')).toBeInTheDocument();
    expect(lastDiscovery()).toEqual({ awsRoleArn: ROLE_ARN });
  });

  it('requires a well-formed role ARN in IAM role mode', async () => {
    renderServiceForm(RdsDiscovery);
    await screen.findByText('rds-mysql');

    clickLabel('IAM role');
    submit();
    expect(await screen.findByText('Required field')).toBeInTheDocument();

    type(
      'text-input-aws-role-arn',
      'arn:aws:iam::123456789012:instance-profile/x'
    );
    submit();
    expect(
      await screen.findByText(
        'Enter an IAM role ARN, for example arn:aws:iam::123456789012:role/RoleName'
      )
    ).toBeInTheDocument();
    expect(discoverRDS).toHaveBeenCalledTimes(1);
  });

  it('requires both keys in access key mode', async () => {
    renderServiceForm(RdsDiscovery);
    await screen.findByText('rds-mysql');

    clickLabel('Access key');
    submit();
    expect(await screen.findAllByText('Required field')).toHaveLength(2);
    expect(discoverRDS).toHaveBeenCalledTimes(1);

    type('text-input-aws-access-key', 'AKIAIOSFODNN7EXAMPLE');
    type('text-input-aws-secret-key', 'secret');
    submit();

    await waitFor(() => expect(discoverRDS).toHaveBeenCalledTimes(2));
    expect(lastDiscovery()).toEqual({
      awsAccessKey: 'AKIAIOSFODNN7EXAMPLE',
      awsSecretKey: 'secret',
    });
  });

  it("sends only the chosen mode's credentials", async () => {
    renderServiceForm(RdsDiscovery);
    await screen.findByText('rds-mysql');

    clickLabel('Access key');
    type('text-input-aws-access-key', 'AKIAIOSFODNN7EXAMPLE');
    type('text-input-aws-secret-key', 'secret');
    discoverWithRole();

    await waitFor(() => expect(discoverRDS).toHaveBeenCalledTimes(2));
    expect(lastDiscovery()).toEqual({ awsRoleArn: ROLE_ARN });
  });

  it("explains a failed discovery with PMM Server's own identity, and the retry", async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    vi.mocked(discoverRDS).mockRejectedValueOnce(new Error('Gateway Timeout'));
    const { unmount } = renderServiceForm(RdsDiscovery, queryClient);
    expect(
      await screen.findByTestId('rds-server-discovery-failed')
    ).toBeInTheDocument();
    unmount();

    vi.mocked(discoverRDS).mockReturnValueOnce(new Promise(() => {}));
    renderServiceForm(RdsDiscovery, queryClient);

    await waitFor(() => expect(discoverRDS).toHaveBeenCalledTimes(2));
    expect(
      screen.getByText("Looking for instances with PMM Server's AWS identity…")
    ).toBeInTheDocument();
    expect(
      screen.queryByTestId('rds-server-discovery-failed')
    ).not.toBeInTheDocument();
    expect(screen.queryByText('No instances found')).not.toBeInTheDocument();
  });

  const rejectDiscovery = (message: string) => {
    const error = new AxiosError('Request failed with status code 500');
    error.response = { status: 500, data: { message } } as AxiosResponse;
    vi.mocked(discoverRDS).mockRejectedValueOnce(error);
  };

  it('explains a discovery without any credentials', async () => {
    renderServiceForm(RdsDiscovery);
    await screen.findByText('rds-mysql');
    // The SDK's error when PMM Server has neither keys nor an IAM role.
    rejectDiscovery(
      'operation error RDS: DescribeDBInstances, get identity: get credentials: failed to refresh cached credentials, no EC2 IMDS role found, operation error ec2imds: GetMetadata, access disabled to EC2 IMDS via client option, or "AWS_EC2_METADATA_DISABLED" environment variable'
    );

    submit();

    await waitFor(() =>
      expect(enqueueSnackbar).toHaveBeenCalledWith(
        'No credentials provided and IAM role is not defined',
        { variant: 'error' }
      )
    );
  });

  it('shows a refused role assumption as it is', async () => {
    renderServiceForm(RdsDiscovery);
    await screen.findByText('rds-mysql');
    const message = `Failed to assume role ${ROLE_ARN}: failed to refresh cached credentials, denied`;
    rejectDiscovery(message);

    submit();

    await waitFor(() =>
      expect(enqueueSnackbar).toHaveBeenCalledWith(message, {
        variant: 'error',
      })
    );
  });

  it('adds a discovered MySQL instance with the discovery credentials', async () => {
    renderServiceForm(RdsDiscovery);
    await screen.findByText('rds-mysql');
    discoverWithRole();
    await waitFor(() => expect(discoverRDS).toHaveBeenCalledTimes(2));

    await startMonitoring('rds-mysql');
    expect(screen.getByTestId('text-input-instance-id')).toHaveValue(
      'rds-mysql'
    );
    expect(screen.getByTestId('text-input-address')).toBeDisabled();
    type('text-input-username', 'admin');
    type('text-input-password', 'secret');
    submit();

    expect(await sentPayload('rds')).toEqual(
      expect.objectContaining({
        engine: DiscoverRDSEngine.mysql,
        instanceId: 'rds-mysql',
        nodeModel: 'db.t3.micro',
        serviceName: 'rds-mysql',
        nodeName: 'rds-mysql',
        address: 'rds-mysql.rds.amazonaws.com',
        port: 3306,
        region: 'eu-west-1',
        az: 'eu-west-1a',
        username: 'admin',
        awsRoleArn: ROLE_ARN,
        rdsExporter: true,
        metricsMode: MetricsMode.pull,
        qanMysqlPerfschema: true,
        tablestatsGroupTableLimit: -1,
        disableBasicMetrics: false,
      })
    );
  });

  it('adds a discovered PostgreSQL instance with pg_stat_statements', async () => {
    renderServiceForm(RdsDiscovery);
    await startMonitoring('rds-pg');

    expect(screen.queryByLabelText('PG Stat Monitor')).not.toBeInTheDocument();
    type('text-input-username', 'postgres');
    type('text-input-password', 'secret');
    submit();

    expect(await sentPayload('rds')).toEqual(
      expect.objectContaining({
        engine: DiscoverRDSEngine.postgresql,
        port: 5432,
        qanPostgresqlPgstatements: true,
        autoDiscoveryLimit: 10,
        maxPostgresqlExporterConnections: undefined,
      })
    );
  });

  it.each(['rds-mysql', 'rds-pg'])(
    "shows PMM Server's 5s cloud default as the %s connection timeout",
    async (instanceId) => {
      renderServiceForm(RdsDiscovery);
      await startMonitoring(instanceId);

      expect(
        screen.getByTestId('text-input-connection-timeout')
      ).toHaveAttribute('placeholder', '5s');
    }
  );

  it('cancels back to the discovered instances without discovering again', async () => {
    renderServiceForm(RdsDiscovery);
    await startMonitoring('rds-mysql');

    fireEvent.click(screen.getByTestId('add-service-cancel'));

    expect(await screen.findByText('rds-pg')).toBeInTheDocument();
    expect(discoverRDS).toHaveBeenCalledTimes(1);
  });

  it('shows the discovery credentials again after cancelling', async () => {
    renderServiceForm(RdsDiscovery);
    await screen.findByText('rds-mysql');
    discoverWithRole();
    await waitFor(() => expect(discoverRDS).toHaveBeenCalledTimes(2));
    await startMonitoring('rds-mysql');

    fireEvent.click(screen.getByTestId('add-service-cancel'));

    expect(await screen.findByTestId('text-input-aws-role-arn')).toHaveValue(
      ROLE_ARN
    );
  });
});
