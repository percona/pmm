import { zodResolver } from '@hookform/resolvers/zod';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { AxiosError } from 'axios';
import type { FC } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { Route, Routes, useLocation } from 'react-router-dom';
import { useNodes } from 'hooks/api/useNodes';
import { AgentStatus, ManagedAgentType } from 'types/agent.types';
import { ManagedNodeType } from 'types/nodes.types';
import { TestWrapper } from 'utils/testWrapper';
import { AdditionalOptions } from '../components/additional-options';
import { ConnectionDetails } from '../components/connection-details';
import { DisableCollectorsField } from '../components/disable-collectors';
import { Labels } from '../components/labels';
import { TlsCertificates } from '../components/tls-certificates';
import { TlsOptions } from '../components/tls-options';
import { ServiceForm } from './ServiceForm';
import {
  COMMON_OPTIONS_DEFAULTS,
  DISABLE_COLLECTORS_DEFAULTS,
  LABELS_DEFAULTS,
  NODE_DEFAULTS,
  TLS_CERTIFICATES_DEFAULTS,
  TLS_DEFAULTS,
} from './ServiceForm.constants';
import {
  commonOptionsSchema,
  connectionSchema,
  disableCollectorsSchema,
  labelsSchema,
  nodeSchema,
  requireCredentialsWithoutTls,
  tlsCertificatesSchema,
  tlsSchema,
} from './ServiceForm.schema';

const { enqueueSnackbar } = vi.hoisted(() => ({ enqueueSnackbar: vi.fn() }));

vi.mock('notistack', () => ({ enqueueSnackbar }));
vi.mock('hooks/api/useNodes', () => ({ useNodes: vi.fn() }));

const schema = z
  .object({
    ...connectionSchema.shape,
    ...nodeSchema.shape,
    ...labelsSchema.shape,
    ...commonOptionsSchema.shape,
    ...tlsSchema.shape,
    ...tlsCertificatesSchema.shape,
    ...disableCollectorsSchema.shape,
  })
  .superRefine(requireCredentialsWithoutTls);

type Values = z.infer<typeof schema>;

const DEFAULTS: Values = {
  serviceName: '',
  address: '',
  port: '3306',
  username: '',
  password: '',
  connectionTimeout: '',
  ...NODE_DEFAULTS,
  ...LABELS_DEFAULTS,
  ...COMMON_OPTIONS_DEFAULTS,
  ...TLS_DEFAULTS,
  ...TLS_CERTIFICATES_DEFAULTS,
  ...DISABLE_COLLECTORS_DEFAULTS,
};

const TestForm: FC<{ onSubmit: (values: Values) => Promise<void> }> = ({
  onSubmit,
}) => {
  const form = useForm<Values>({
    resolver: zodResolver(schema),
    mode: 'onTouched',
    defaultValues: DEFAULTS,
  });

  return (
    <ServiceForm form={form} serviceTypeLabel="MySQL" onSubmit={onSubmit}>
      <ConnectionDetails requiresCredentials withNodes />
      <Labels />
      <AdditionalOptions>
        <TlsOptions certificates={<TlsCertificates />} />
        <DisableCollectorsField />
      </AdditionalOptions>
    </ServiceForm>
  );
};

const LocationProbe = () => (
  <span data-testid="location-probe">{useLocation().pathname}</span>
);

const renderForm = (onSubmit = vi.fn().mockResolvedValue(undefined)) => {
  render(
    <TestWrapper routerProps={{ initialEntries: ['/add-instance/mysql'] }}>
      <LocationProbe />
      <Routes>
        <Route path="*" element={<TestForm onSubmit={onSubmit} />} />
      </Routes>
    </TestWrapper>
  );
  return onSubmit;
};

const type = (testId: string, value: string) =>
  fireEvent.change(screen.getByTestId(testId), { target: { value } });

const turnOnTls = () =>
  fireEvent.click(screen.getByLabelText('Use TLS for database connections'));

const fillRequired = () => {
  type('text-input-address', 'db.example.com');
  type('text-input-username', 'pmm');
  type('text-input-password', 'secret');
};

describe('ServiceForm', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(useNodes).mockReturnValue({
      data: {
        nodes: [
          {
            nodeId: 'pmm-server',
            nodeName: 'pmm-server',
            nodeType: ManagedNodeType.generic,
            address: '',
            isPmmServerNode: true,
            isPmmInternalNode: false,
            agents: [
              {
                agentId: 'pmm-server',
                agentType: ManagedAgentType.pmmAgent,
                status: AgentStatus.running,
                isConnected: true,
              },
            ],
          },
        ],
      },
      isLoading: false,
    } as ReturnType<typeof useNodes>);
  });

  it('renders the shared sections', () => {
    renderForm();

    for (const testId of [
      'add-service-form',
      'text-input-service-name',
      'text-input-address',
      'text-input-port',
      'text-input-username',
      'text-input-password',
      'text-input-connection-timeout',
      'select-input-node-id',
      'select-input-pmm-agent-id',
      'text-input-environment',
      'text-input-cluster',
      'text-input-replication-set',
      'text-input-region',
      'text-input-az',
      'text-input-custom-labels',
      'switch-input-skip-connection-check',
      'switch-input-tls',
      'text-input-disable-collectors',
    ]) {
      expect(screen.getByTestId(testId)).toBeInTheDocument();
    }
  });

  it('shows the TLS certificates and verification only once TLS is on', async () => {
    renderForm();

    expect(screen.queryByTestId('text-input-tls-ca')).not.toBeInTheDocument();
    expect(
      screen.queryByTestId('switch-input-tls-skip-verify')
    ).not.toBeInTheDocument();

    turnOnTls();

    expect(await screen.findByTestId('text-input-tls-ca')).toBeInTheDocument();
    expect(screen.getByTestId('text-input-tls-key')).toBeInTheDocument();
    expect(screen.getByTestId('text-input-tls-cert')).toBeInTheDocument();
    expect(
      screen.getByTestId('switch-input-tls-skip-verify')
    ).toBeInTheDocument();
  });

  it('blocks submitting without the required fields', async () => {
    const onSubmit = renderForm();

    fireEvent.click(screen.getByTestId('add-service-submit'));

    await waitFor(() =>
      expect(screen.getByTestId('text-input-address')).toHaveAttribute(
        'aria-invalid',
        'true'
      )
    );
    for (const testId of ['text-input-username', 'text-input-password']) {
      expect(screen.getByTestId(testId)).toHaveAttribute(
        'aria-invalid',
        'true'
      );
    }
    expect(screen.getAllByText('Required field')).toHaveLength(3);
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it('marks the required fields, the credentials only without TLS', () => {
    renderForm();

    for (const testId of [
      'text-input-address',
      'text-input-port',
      'text-input-username',
      'text-input-password',
    ]) {
      expect(screen.getByTestId(testId)).toBeRequired();
    }
    expect(
      screen.getByTestId('text-input-connection-timeout')
    ).not.toBeRequired();

    turnOnTls();

    expect(screen.getByTestId('text-input-username')).not.toBeRequired();
    expect(screen.getByTestId('text-input-password')).not.toBeRequired();
  });

  it('does not ask for credentials with TLS on', async () => {
    const onSubmit = renderForm();

    type('text-input-address', 'db.example.com');
    turnOnTls();
    fireEvent.click(screen.getByTestId('add-service-submit'));

    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
  });

  it('reports the added service and returns to the inventory', async () => {
    const onSubmit = renderForm();

    fillRequired();
    type('text-input-custom-labels', 'team:db');
    fireEvent.click(screen.getByTestId('add-service-submit'));

    await waitFor(() =>
      expect(screen.getByTestId('location-probe')).toHaveTextContent(
        '/graph/inventory'
      )
    );
    expect(onSubmit.mock.calls[0][0]).toEqual(
      expect.objectContaining({
        address: 'db.example.com',
        nodeId: 'pmm-server',
        pmmAgentId: 'pmm-server',
        customLabels: 'team:db',
      })
    );
    expect(enqueueSnackbar).toHaveBeenCalledWith(
      'Service “db.example.com” added to your inventory. Your MySQL service instance is now ready to be monitored.',
      { variant: 'success' }
    );
  });

  it('stays on the form when the server rejects the service', async () => {
    const error = new AxiosError('failed');
    error.response = { status: 400 } as AxiosError['response'];
    error.notified = true;
    const onSubmit = renderForm(vi.fn().mockRejectedValue(error));

    fillRequired();
    fireEvent.click(screen.getByTestId('add-service-submit'));

    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    await waitFor(() =>
      expect(screen.getByTestId('add-service-submit')).toBeEnabled()
    );
    // the API layer has already reported it
    expect(enqueueSnackbar).not.toHaveBeenCalled();
    expect(screen.getByTestId('location-probe')).toHaveTextContent(
      '/add-instance/mysql'
    );
  });

  it('reports a failure the API layer did not', async () => {
    renderForm(vi.fn().mockRejectedValue(new Error('Network Error')));

    fillRequired();
    fireEvent.click(screen.getByTestId('add-service-submit'));

    await waitFor(() =>
      expect(enqueueSnackbar).toHaveBeenCalledWith('Network Error', {
        variant: 'error',
      })
    );
    expect(screen.getByTestId('location-probe')).toHaveTextContent(
      '/add-instance/mysql'
    );
  });

  it('reports a rejected response the API layer kept quiet about', async () => {
    const error = new AxiosError('Request failed with status code 429');
    error.response = { status: 429 } as AxiosError['response'];
    renderForm(vi.fn().mockRejectedValue(error));

    fillRequired();
    fireEvent.click(screen.getByTestId('add-service-submit'));

    await waitFor(() =>
      expect(enqueueSnackbar).toHaveBeenCalledWith(
        'Request failed with status code 429',
        { variant: 'error' }
      )
    );
  });

  it('prefers the server message for a failure the API layer kept quiet about', async () => {
    const error = new AxiosError('Request failed with status code 429');
    error.response = {
      status: 429,
      data: { message: 'Too many requests, retry in 30s' },
    } as AxiosError['response'];
    renderForm(vi.fn().mockRejectedValue(error));

    fillRequired();
    fireEvent.click(screen.getByTestId('add-service-submit'));

    await waitFor(() =>
      expect(enqueueSnackbar).toHaveBeenCalledWith(
        'Too many requests, retry in 30s',
        { variant: 'error' }
      )
    );
  });

  it('cancels back to the service type selection', () => {
    renderForm();

    expect(screen.getByTestId('add-service-cancel')).toHaveAttribute(
      'href',
      '/add-instance'
    );
  });
});
