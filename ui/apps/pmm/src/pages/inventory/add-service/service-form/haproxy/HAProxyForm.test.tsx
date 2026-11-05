import { screen } from '@testing-library/react';
import { MetricsMode } from 'types/services.types';
import { NodeType } from 'types/nodes.types';
import {
  mockAddService,
  renderServiceForm,
  sentPayload,
  submit,
  type,
} from '../ServiceForm.testUtils';
import { HAProxyForm } from './HAProxyForm';

vi.mock('api/services', () => ({ addService: vi.fn() }));
vi.mock('notistack', () => ({ enqueueSnackbar: vi.fn() }));

describe('HAProxyForm', () => {
  beforeEach(() => {
    mockAddService();
  });

  it('adds an HAProxy exporter scraped from a remote node', async () => {
    renderServiceForm(HAProxyForm);

    type('text-input-address', ' haproxy.example.com ');
    type('text-input-username', ' stats ');
    type('text-input-password', ' secret ');
    submit();

    expect(await sentPayload('haproxy')).toEqual({
      serviceName: 'haproxy.example.com',
      address: 'haproxy.example.com',
      environment: '',
      cluster: '',
      replicationSet: '',
      customLabels: undefined,
      addNode: { nodeName: 'haproxy.example.com', nodeType: NodeType.remote },
      metricsMode: MetricsMode.pull,
      skipConnectionCheck: false,
      tlsSkipVerify: false,
      listenPort: 8404,
      username: 'stats',
      password: 'secret',
    });
  });

  it('does not offer skipping TLS verification', () => {
    renderServiceForm(HAProxyForm);

    expect(
      screen.queryByTestId('switch-input-tls-skip-verify')
    ).not.toBeInTheDocument();
  });
});
