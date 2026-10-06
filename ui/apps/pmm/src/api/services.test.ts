import type { AxiosResponse, InternalAxiosRequestConfig } from 'axios';
import { NodeType } from 'types/nodes.types';
import {
  type AddMySQLServiceParams,
  DiscoverAzureDatabaseType,
  MetricsMode,
} from 'types/services.types';
import { api } from './api';
import {
  addAzureService,
  addService,
  discoverAzure,
  discoverRDS,
} from './services';

const requests: InternalAxiosRequestConfig[] = [];
const originalAdapter = api.defaults.adapter;

const sentBody = () => JSON.parse(requests[0].data);

const MYSQL: AddMySQLServiceParams = {
  serviceName: 'mysql-1',
  address: 'db.example.com',
  port: 3306,
  pmmAgentId: 'pmm-server',
  username: 'root',
  qanMysqlPerfschema: true,
  tablestatsGroupTableLimit: -1,
  metricsMode: MetricsMode.push,
  addNode: {
    nodeName: 'mysql-1',
    nodeType: NodeType.remote,
    customLabels: { nodeLabel: 'a' },
  },
  customLabels: { myLabel: 'a', 'my-label': 'b', ENV: 'prod' },
  extraDsnParams: { allowCleartextPasswords: '1' },
};

describe('services api', () => {
  beforeEach(() => {
    requests.length = 0;
    api.defaults.adapter = async (config) => {
      requests.push(config);
      return { data: {}, status: 200, config } as AxiosResponse;
    };
  });

  afterEach(() => {
    api.defaults.adapter = originalAdapter;
  });

  it('adds a service wrapped in its type with snake_case fields', async () => {
    await addService({ mysql: MYSQL });

    expect(requests[0].url).toBe('/management/services');
    expect(sentBody()).toEqual({
      mysql: expect.objectContaining({
        service_name: 'mysql-1',
        pmm_agent_id: 'pmm-server',
        qan_mysql_perfschema: true,
        tablestats_group_table_limit: -1,
        metrics_mode: 'METRICS_MODE_PUSH',
        add_node: {
          node_name: 'mysql-1',
          node_type: 'NODE_TYPE_REMOTE_NODE',
          custom_labels: { nodeLabel: 'a' },
        },
      }),
    });
  });

  it('keeps label and DSN parameter keys as entered', async () => {
    await addService({ mysql: MYSQL });

    const { mysql } = sentBody();
    expect(mysql.custom_labels).toEqual({
      myLabel: 'a',
      'my-label': 'b',
      ENV: 'prod',
    });
    expect(mysql.extra_dsn_params).toEqual({ allowCleartextPasswords: '1' });
  });

  it('does not change the payload it was given', async () => {
    const payload = { mysql: MYSQL };

    await addService(payload);

    expect(Object.getPrototypeOf(payload.mysql.customLabels)).toBe(
      Object.prototype
    );
  });

  it('adds an Azure database without wrapping the body', async () => {
    await addAzureService({
      region: 'eastus',
      instanceId: 'azure-1',
      address: 'azure-1.mysql.database.azure.com',
      port: 3306,
      username: 'admin',
      azureClientId: 'client',
      azureClientSecret: 'secret',
      azureTenantId: 'tenant',
      azureSubscriptionId: 'subscription',
      azureResourceGroup: 'group',
      type: DiscoverAzureDatabaseType.mysql,
      customLabels: { myLabel: 'a' },
    });

    expect(requests[0].url).toBe('/management/services/azure');
    expect(sentBody()).toEqual(
      expect.objectContaining({
        instance_id: 'azure-1',
        azure_resource_group: 'group',
        type: 'DISCOVER_AZURE_DATABASE_TYPE_MYSQL',
        custom_labels: { myLabel: 'a' },
      })
    );
  });

  it('discovers RDS instances without the default error notification', async () => {
    await discoverRDS({ awsAccessKey: 'key', awsSecretKey: 'secret' });

    expect(requests[0].url).toBe('/management/services:discoverRDS');
    expect(requests[0].disableNotifications).toBe(true);
    expect(sentBody()).toEqual({
      aws_access_key: 'key',
      aws_secret_key: 'secret',
    });
  });

  it('discovers Azure instances', async () => {
    await discoverAzure({
      azureClientId: 'client',
      azureClientSecret: 'secret',
      azureTenantId: 'tenant',
      azureSubscriptionId: 'subscription',
    });

    expect(requests[0].url).toBe('/management/services:discoverAzure');
    expect(requests[0].disableNotifications).toBeUndefined();
    expect(sentBody()).toEqual({
      azure_client_id: 'client',
      azure_client_secret: 'secret',
      azure_tenant_id: 'tenant',
      azure_subscription_id: 'subscription',
    });
  });
});
