import { screen } from '@testing-library/react';
import { addService } from 'api/services';
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
});
