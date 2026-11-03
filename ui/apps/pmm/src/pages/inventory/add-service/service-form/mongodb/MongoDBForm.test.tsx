import { screen, waitFor } from '@testing-library/react';
import { addService } from 'api/services';
import {
  clickLabel,
  mockAddService,
  mockNodes,
  renderServiceForm,
  sentPayload,
  submit,
  type,
} from '../ServiceForm.testUtils';
import { MongoDBForm } from './MongoDBForm';

vi.mock('api/services', () => ({ addService: vi.fn() }));
vi.mock('hooks/api/useNodes', () => ({ useNodes: vi.fn() }));
vi.mock('notistack', () => ({ enqueueSnackbar: vi.fn() }));

describe('MongoDBForm', () => {
  beforeEach(() => {
    mockAddService();
    mockNodes();
  });

  it('adds a MongoDB service with credentials', async () => {
    renderServiceForm(MongoDBForm);

    type('text-input-address', 'mongo.example.com');
    type('text-input-username', 'pmm');
    type('text-input-password', 'secret');
    clickLabel('Use QAN MongoDB Profiler');
    submit();

    const mongodb = await sentPayload('mongodb');
    expect(mongodb).toEqual(
      expect.objectContaining({
        port: 27017,
        username: 'pmm',
        tls: false,
        qanMongodbProfiler: true,
      })
    );
    expect(mongodb).not.toHaveProperty('authenticationMechanism');
  });

  it('authenticates with the client certificate under TLS', async () => {
    renderServiceForm(MongoDBForm);

    type('text-input-address', 'mongo.example.com');
    clickLabel('Use TLS for database connections');
    await screen.findByTestId('text-input-tls-certificate-key');
    type('text-input-tls-certificate-key-file-password', 'keypass');
    type('text-input-tls-certificate-key', 'KEY');
    type('text-input-tls-ca', 'CA');
    submit();

    expect(await sentPayload('mongodb')).toEqual(
      expect.objectContaining({
        tls: true,
        tlsCertificateKeyFilePassword: 'keypass',
        tlsCertificateKey: 'KEY',
        tlsCa: 'CA',
        authenticationMechanism: 'MONGODB-X509',
      })
    );
  });

  it('requires credentials without TLS', async () => {
    renderServiceForm(MongoDBForm);

    type('text-input-address', 'mongo.example.com');
    submit();

    await waitFor(() =>
      expect(screen.getByTestId('text-input-username')).toHaveAttribute(
        'aria-invalid',
        'true'
      )
    );
    expect(addService).not.toHaveBeenCalled();
  });
});
