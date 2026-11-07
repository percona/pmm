import { fireEvent, screen, waitFor } from '@testing-library/react';
import { addService } from 'api/services';
import { enqueueSnackbar } from 'notistack';
import { MetricsMode } from 'types/services.types';
import { InventoryNodeType } from 'types/nodes.types';
import {
  clickLabel,
  mockAddService,
  renderServiceForm,
  sentPayload,
  submit,
  type,
} from '../ServiceForm.testUtils';
import { ExternalForm } from './ExternalForm';

vi.mock('api/services', () => ({ addService: vi.fn() }));
vi.mock('notistack', () => ({ enqueueSnackbar: vi.fn() }));

describe('ExternalForm', () => {
  beforeEach(() => {
    mockAddService();
  });

  it('adds an exporter entered field by field, over HTTP by default', async () => {
    renderServiceForm(ExternalForm);

    type('text-input-group', 'redis');
    type('text-input-address', 'redis-exporter');
    type('text-input-port', '9121');
    type('text-input-metrics-path', '/metrics');
    submit();

    expect(await sentPayload('external')).toEqual({
      serviceName: 'redis-exporter',
      address: 'redis-exporter',
      environment: '',
      cluster: '',
      replicationSet: '',
      customLabels: undefined,
      addNode: {
        nodeName: 'redis-exporter',
        nodeType: InventoryNodeType.remote,
      },
      metricsMode: MetricsMode.pull,
      skipConnectionCheck: false,
      tlsSkipVerify: false,
      scheme: 'http',
      listenPort: 9121,
      metricsPath: '/metrics',
      username: '',
      password: '',
      group: 'redis',
    });
  });

  it('adds an exporter parsed from its URL', async () => {
    renderServiceForm(ExternalForm);

    clickLabel('Parse from URL string');
    type('text-input-url', 'http://redis-exporter:9121/metrics');
    submit();

    expect(await sentPayload('external')).toEqual(
      expect.objectContaining({
        serviceName: 'redis-exporter',
        scheme: 'http',
        address: 'redis-exporter',
        listenPort: 9121,
        metricsPath: '/metrics',
      })
    );
  });

  it('checks only the fields of the chosen mode', async () => {
    renderServiceForm(ExternalForm);

    clickLabel('Parse from URL string');
    type('text-input-url', 'not a url');
    submit();

    expect(await screen.findByText('Invalid URL string')).toBeInTheDocument();
    expect(screen.queryByText('Required field')).not.toBeInTheDocument();
    expect(addService).not.toHaveBeenCalled();
  });

  it('reports the added exporter as an external service', async () => {
    renderServiceForm(ExternalForm);

    type('text-input-address', 'redis-exporter');
    type('text-input-port', '9121');
    submit();

    await waitFor(() =>
      expect(enqueueSnackbar).toHaveBeenCalledWith(
        'Service “redis-exporter” added to your inventory. Your external service instance is now ready to be monitored.',
        { variant: 'success' }
      )
    );
  });

  it('offers skipping TLS verification only over HTTPS', async () => {
    renderServiceForm(ExternalForm);
    const skipVerify = () =>
      screen.queryByTestId('switch-input-tls-skip-verify');

    expect(skipVerify()).not.toBeInTheDocument();
    clickLabel('HTTPS');
    expect(skipVerify()).toBeInTheDocument();
    clickLabel('Skip TLS certificate and hostname validation');
    clickLabel('HTTP');
    expect(skipVerify()).not.toBeInTheDocument();

    clickLabel('Parse from URL string');
    type('text-input-url', 'http://redis-exporter:9121/metrics');
    expect(skipVerify()).not.toBeInTheDocument();
    type('text-input-url', 'https://redis-exporter:9121/metrics');
    expect(skipVerify()).toBeInTheDocument();
    type('text-input-url', 'http://redis-exporter:9121/metrics');
    submit();

    expect(await sentPayload('external')).toEqual(
      expect.objectContaining({ scheme: 'http', tlsSkipVerify: false })
    );
  });

  it('starts at the HTTP port and follows the scheme', () => {
    renderServiceForm(ExternalForm);
    const port = () => screen.getByTestId('text-input-port');

    expect(port()).toHaveValue('80');
    clickLabel('HTTPS');
    expect(port()).toHaveValue('443');

    type('text-input-port', '9121');
    clickLabel('HTTP');
    expect(port()).toHaveValue('9121');
  });

  it('explains the scheme in a tooltip', async () => {
    renderServiceForm(ExternalForm);

    const schema = screen.getByText('Schema');
    fireEvent.mouseOver(
      schema.parentElement!.querySelector('[data-testid="InfoOutlinedIcon"]')!
    );

    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      'Used version of protocol'
    );
  });

  it('keeps a parsed URL when switching to manual entry', () => {
    renderServiceForm(ExternalForm);

    clickLabel('Parse from URL string');
    type('text-input-url', 'http://user:pass@redis-exporter:9121/metrics2');
    clickLabel('Set manually');

    expect(screen.getByLabelText('HTTP')).toBeChecked();
    expect(screen.getByTestId('text-input-address')).toHaveValue(
      'redis-exporter'
    );
    expect(screen.getByTestId('text-input-port')).toHaveValue('9121');
    expect(screen.getByTestId('text-input-metrics-path')).toHaveValue(
      '/metrics2'
    );
    expect(screen.getByTestId('text-input-username')).toHaveValue('user');
    expect(screen.getByTestId('text-input-password')).toHaveValue('pass');
  });

  it('keeps the port a parsed URL names, even a default one', () => {
    renderServiceForm(ExternalForm);

    clickLabel('Parse from URL string');
    type('text-input-url', 'https://exporter:80/metrics');
    clickLabel('Set manually');

    expect(screen.getByLabelText('HTTPS')).toBeChecked();
    expect(screen.getByTestId('text-input-port')).toHaveValue('80');

    clickLabel('HTTP');
    expect(screen.getByTestId('text-input-port')).toHaveValue('80');
    clickLabel('HTTPS');
    expect(screen.getByTestId('text-input-port')).toHaveValue('443');
  });

  it('rejects a URL whose credentials do not decode', () => {
    renderServiceForm(ExternalForm);

    clickLabel('Parse from URL string');
    type('text-input-url', 'http://user:pa%zz@exporter:9100/metrics');
    clickLabel('Set manually');

    expect(screen.getByTestId('text-input-address')).toHaveValue('');
  });
});
