import { fireEvent, screen, waitFor } from '@testing-library/react';
import { addService } from 'api/services';
import { enqueueSnackbar } from 'notistack';
import { MetricsMode } from 'types/services.types';
import { NodeType } from 'types/nodes.types';
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

  it('adds an exporter entered field by field, over HTTPS by default', async () => {
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
      addNode: { nodeName: 'redis-exporter', nodeType: NodeType.remote },
      metricsMode: MetricsMode.pull,
      skipConnectionCheck: false,
      tlsSkipVerify: false,
      scheme: 'https',
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

  it('starts at the HTTPS port and follows the scheme', () => {
    renderServiceForm(ExternalForm);
    const port = () => screen.getByTestId('text-input-port');

    expect(port()).toHaveValue('443');
    clickLabel('HTTP');
    expect(port()).toHaveValue('80');

    type('text-input-port', '9121');
    clickLabel('HTTPS');
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
});
