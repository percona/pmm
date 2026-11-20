import { screen } from '@testing-library/react';
import { addService } from 'api/services';
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
import { ProxySQLForm } from './ProxySQLForm';

vi.mock('api/services', () => ({ addService: vi.fn() }));
vi.mock('api/nodes', () => ({ listNodes: vi.fn() }));
vi.mock('notistack', () => ({ enqueueSnackbar: vi.fn() }));

describe('ProxySQLForm', () => {
  beforeEach(() => {
    mockAddService();
    mockNodes();
  });

  it('adds a ProxySQL service', async () => {
    renderServiceForm(ProxySQLForm);
    await waitForNodePreselect();

    type('text-input-address', 'proxysql.example.com');
    type('text-input-username', 'admin');
    type('text-input-password', 'admin');
    clickLabel('Use TLS for database connections');
    type('text-input-disable-collectors', 'mysql_connection_pool');
    submit();

    expect(await sentPayload('proxysql')).toEqual(
      expect.objectContaining({
        address: 'proxysql.example.com',
        port: 6032,
        tls: true,
        disableCollectors: ['mysql_connection_pool'],
      })
    );
  });

  it('still asks for the password with TLS on', async () => {
    renderServiceForm(ProxySQLForm);
    await waitForNodePreselect();

    type('text-input-address', 'proxysql.example.com');
    type('text-input-username', 'admin');
    clickLabel('Use TLS for database connections');

    expect(screen.getByTestId('text-input-password')).toBeRequired();
    submit();

    expect(await screen.findByText('Required field')).toBeInTheDocument();
    expect(addService).not.toHaveBeenCalled();
  });
});
