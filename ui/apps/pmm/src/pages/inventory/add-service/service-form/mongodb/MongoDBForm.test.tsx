import { screen } from '@testing-library/react';
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
import { MongoDBForm } from './MongoDBForm';

vi.mock('api/services', () => ({ addService: vi.fn() }));
vi.mock('api/nodes', () => ({ listNodes: vi.fn() }));
vi.mock('notistack', () => ({ enqueueSnackbar: vi.fn() }));

describe('MongoDBForm', () => {
  beforeEach(() => {
    mockAddService();
    mockNodes();
  });

  it('adds a MongoDB service with credentials', async () => {
    renderServiceForm(MongoDBForm);
    await waitForNodePreselect();

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
    await waitForNodePreselect();

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

  it('adds a MongoDB without authentication', async () => {
    renderServiceForm(MongoDBForm);
    await waitForNodePreselect();

    type('text-input-address', 'mongo.example.com');
    submit();

    const mongodb = await sentPayload('mongodb');
    expect(mongodb).toEqual(
      expect.objectContaining({ tls: false, username: '', password: '' })
    );
    expect(mongodb).not.toHaveProperty('authenticationMechanism');
  });

  it('keeps password authentication over TLS when a username is given', async () => {
    renderServiceForm(MongoDBForm);
    await waitForNodePreselect();

    type('text-input-address', 'mongo.example.com');
    type('text-input-username', 'pmm');
    type('text-input-password', 'secret');
    clickLabel('Use TLS for database connections');
    submit();

    const mongodb = await sentPayload('mongodb');
    expect(mongodb).toEqual(
      expect.objectContaining({ tls: true, username: 'pmm' })
    );
    expect(mongodb).not.toHaveProperty('authenticationMechanism');
  });
});
