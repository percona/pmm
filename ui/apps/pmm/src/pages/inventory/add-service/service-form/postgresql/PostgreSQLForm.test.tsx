import { fireEvent, screen } from '@testing-library/react';
import { MetricsMode } from 'types/services.types';
import { InventoryNodeType } from 'types/nodes.types';
import {
  clickLabel,
  mockAddService,
  mockNodes,
  renderServiceForm,
  sentPayload,
  submit,
  type,
  waitForNodePreselect,
} from '../ServiceForm.testUtils';
import { PostgreSQLForm } from './PostgreSQLForm';

vi.mock('api/services', () => ({ addService: vi.fn() }));
vi.mock('api/nodes', () => ({ listNodes: vi.fn() }));
vi.mock('notistack', () => ({ enqueueSnackbar: vi.fn() }));

const fillRequired = () => {
  type('text-input-address', 'pg.example.com');
  type('text-input-username', 'pmm');
  type('text-input-password', 'secret');
};

describe('PostgreSQLForm', () => {
  beforeEach(() => {
    mockAddService();
    mockNodes();
  });

  it('adds a remote PostgreSQL service with the default options', async () => {
    renderServiceForm(PostgreSQLForm);
    await waitForNodePreselect();

    fillRequired();
    submit();

    expect(await sentPayload('postgresql')).toEqual({
      serviceName: 'pg.example.com',
      address: 'pg.example.com',
      port: 5432,
      pmmAgentId: 'pmm-server-agent',
      username: 'pmm',
      password: 'secret',
      connectionTimeout: undefined,
      skipConnectionCheck: false,
      tlsSkipVerify: false,
      metricsMode: MetricsMode.pull,
      environment: '',
      cluster: '',
      replicationSet: '',
      customLabels: undefined,
      addNode: {
        nodeName: 'pg.example.com',
        nodeType: InventoryNodeType.remote,
      },
      database: undefined,
      tls: false,
      qanPostgresqlPgstatementsAgent: false,
      qanPostgresqlPgstatmonitorAgent: true,
      disableQueryExamples: false,
      disableCommentsParsing: true,
      maxQueryLength: undefined,
      disableCollectors: undefined,
      autoDiscoveryLimit: 10,
      maxExporterConnections: undefined,
    });
  });

  it('sends the chosen tracking and limits', async () => {
    renderServiceForm(PostgreSQLForm);
    await waitForNodePreselect();

    fillRequired();
    type('text-input-database', 'app');
    clickLabel('PG Stat Statements');
    clickLabel('Custom');
    type('text-input-auto-discovery-limit', '25');
    fireEvent.click(screen.getAllByLabelText('Enabled')[1]);
    type('text-input-max-exporter-connections', '3');
    submit();

    expect(await sentPayload('postgresql')).toEqual(
      expect.objectContaining({
        database: 'app',
        qanPostgresqlPgstatementsAgent: true,
        qanPostgresqlPgstatmonitorAgent: false,
        autoDiscoveryLimit: 25,
        maxExporterConnections: 3,
      })
    );
  });

  it('turns auto-discovery off with the disabled limit', async () => {
    renderServiceForm(PostgreSQLForm);
    await waitForNodePreselect();

    fillRequired();
    fireEvent.click(screen.getAllByLabelText('Disabled')[0]);
    submit();

    expect(await sentPayload('postgresql')).toEqual(
      expect.objectContaining({ autoDiscoveryLimit: -1 })
    );
  });

  it('shows the limits each option sends', async () => {
    renderServiceForm(PostgreSQLForm);
    await waitForNodePreselect();

    expect(screen.getByTestId('text-input-auto-discovery-limit')).toHaveValue(
      '10'
    );
    expect(
      screen.getByTestId('text-input-auto-discovery-limit')
    ).toBeDisabled();
    expect(
      screen.getByTestId('text-input-max-exporter-connections')
    ).toBeDisabled();
  });
});
