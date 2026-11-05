import { screen, waitFor } from '@testing-library/react';
import { addService } from 'api/services';
import { MetricsMode } from 'types/services.types';
import { NodeType } from 'types/nodes.types';
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
import { MySQLForm } from './MySQLForm';

vi.mock('api/services', () => ({ addService: vi.fn() }));
vi.mock('api/nodes', () => ({ listNodes: vi.fn() }));
vi.mock('notistack', () => ({ enqueueSnackbar: vi.fn() }));

const fillRequired = () => {
  type('text-input-address', 'mysql.example.com');
  type('text-input-username', 'pmm');
  type('text-input-password', 'secret');
};

describe('MySQLForm', () => {
  beforeEach(() => {
    mockAddService();
    mockNodes();
  });

  it('adds a remote MySQL service with the default options', async () => {
    renderServiceForm(MySQLForm);
    await waitForNodePreselect();

    fillRequired();
    submit();

    expect(await sentPayload('mysql')).toEqual({
      serviceName: 'mysql.example.com',
      address: 'mysql.example.com',
      port: 3306,
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
      addNode: { nodeName: 'mysql.example.com', nodeType: NodeType.remote },
      tls: false,
      qanMysqlPerfschema: true,
      disableQueryExamples: false,
      disableCommentsParsing: true,
      maxQueryLength: undefined,
      extraDsnParams: undefined,
      disableCollectors: undefined,
      tablestatsGroupTableLimit: -1,
    });
  });

  it('sends the typed options', async () => {
    renderServiceForm(MySQLForm);
    await waitForNodePreselect();

    fillRequired();
    type('text-input-max-query-length', '2048');
    type('text-input-extra-dsn-params', 'allowCleartextPasswords:1');
    type('text-input-disable-collectors', 'binlog_size, engine_innodb');
    type('text-input-custom-labels', 'team:db');
    clickLabel('Custom');
    type('text-input-tablestats-group-table-limit', '500');
    submit();

    expect(await sentPayload('mysql')).toEqual(
      expect.objectContaining({
        maxQueryLength: 2048,
        extraDsnParams: { allowCleartextPasswords: '1' },
        disableCollectors: ['binlog_size', 'engine_innodb'],
        customLabels: { team: 'db' },
        tablestatsGroupTableLimit: 500,
      })
    );
  });

  it('sends the certificates only while TLS is on', async () => {
    renderServiceForm(MySQLForm);
    await waitForNodePreselect();

    type('text-input-address', 'mysql.example.com');
    clickLabel('Use TLS for database connections');
    await screen.findByTestId('text-input-tls-ca');
    type('text-input-tls-ca', 'CA');
    clickLabel('Use TLS for database connections');
    type('text-input-username', 'pmm');
    type('text-input-password', 'secret');
    submit();

    const mysql = await sentPayload('mysql');
    expect(mysql).toEqual(expect.objectContaining({ tls: false }));
    expect(mysql).not.toHaveProperty('tlsCa');
  });

  it('requires a limit for custom table statistics', async () => {
    renderServiceForm(MySQLForm);
    await waitForNodePreselect();

    fillRequired();
    clickLabel('Custom');
    type('text-input-tablestats-group-table-limit', '');
    submit();

    await waitFor(() =>
      expect(
        screen.getByTestId('text-input-tablestats-group-table-limit')
      ).toHaveAttribute('aria-invalid', 'true')
    );
    expect(addService).not.toHaveBeenCalled();
  });

  it('ignores a custom limit once a preset is chosen again', async () => {
    renderServiceForm(MySQLForm);
    await waitForNodePreselect();

    fillRequired();
    clickLabel('Custom');
    type('text-input-tablestats-group-table-limit', '1.5');
    clickLabel('Default');
    submit();

    expect(await sentPayload('mysql')).toEqual(
      expect.objectContaining({ tablestatsGroupTableLimit: 1000 })
    );
  });

  it('drops TLS verification skipping once TLS is turned off', async () => {
    renderServiceForm(MySQLForm);
    await waitForNodePreselect();
    fillRequired();

    clickLabel('Use TLS for database connections');
    clickLabel('Skip TLS certificate and hostname validation');
    clickLabel('Use TLS for database connections');

    expect(
      screen.queryByTestId('switch-input-tls-skip-verify')
    ).not.toBeInTheDocument();
    submit();

    expect(await sentPayload('mysql')).toEqual(
      expect.objectContaining({ tls: false, tlsSkipVerify: false })
    );
  });

  it('shows the table statistics limit each option sends', async () => {
    renderServiceForm(MySQLForm);
    await waitForNodePreselect();
    const limit = () =>
      screen.getByTestId('text-input-tablestats-group-table-limit');

    expect(limit()).toHaveValue('-1');
    expect(limit()).toBeDisabled();

    clickLabel('Default');
    expect(limit()).toHaveValue('1000');
    expect(limit()).toBeDisabled();

    clickLabel('Custom');
    expect(limit()).toBeEnabled();
  });
});
