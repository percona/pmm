import {
  AxiosError,
  type AxiosResponse,
  type InternalAxiosRequestConfig,
} from 'axios';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { AgentStatus, AgentType } from 'types/agent.types';
import { NodeType } from 'types/nodes.types';
import { ManagedServiceType, ServiceStatus } from 'types/services.types';
import { api } from './api';
// registers the error notifier on the api instance
import './api.interceptors';
import {
  findRdsExporter,
  getIfExists,
  toBaseService,
  toManagedAgent,
  toManagedNode,
  toManagedService,
  type NodePayload,
  type ServicePayload,
} from './inventory.utils';

const { enqueueSnackbar } = vi.hoisted(() => ({ enqueueSnackbar: vi.fn() }));

vi.mock('notistack', () => ({ enqueueSnackbar }));

const SERVICE_PAYLOAD: ServicePayload = {
  service_id: 'service-1',
  service_type: ManagedServiceType.mysql,
  service_name: 'mysql-1',
  database_name: '',
  node_id: 'node-1',
  node_name: 'node-1',
  environment: 'prod',
  cluster: 'cluster-1',
  replication_set: '',
  custom_labels: { my_label: 'value', 'app.kubernetes.io/name': 'db' },
  external_group: '',
  address: '10.0.0.1',
  port: 3306,
  socket: '',
  version: '8.0.36',
  status: ServiceStatus.up,
  created_at: '2026-01-01T00:00:00Z',
  agents: [
    {
      agent_id: 'agent-1',
      agent_type: AgentType.mysqldExporter,
      status: AgentStatus.running,
      is_connected: false,
      disabled: false,
    },
  ],
};

const NODE_PAYLOAD: NodePayload = {
  node_id: 'node-1',
  node_type: NodeType.generic,
  node_name: 'node-1',
  address: '10.0.0.1',
  status: ServiceStatus.up,
  custom_labels: { rack_id: '7' },
  is_pmm_server_node: false,
  machine_id: 'machine-1',
  distro: '',
  agents: [
    {
      agent_id: 'pmm-agent-1',
      agent_type: AgentType.pmmAgent,
      status: '',
      is_connected: true,
    },
  ],
  services: [
    {
      service_id: 'service-1',
      service_type: ManagedServiceType.mysql,
      service_name: 'mysql-1',
    },
  ],
};

describe('toManagedService', () => {
  it('keeps custom label keys as the API returns them', () => {
    expect(toManagedService(SERVICE_PAYLOAD).customLabels).toEqual({
      my_label: 'value',
      'app.kubernetes.io/name': 'db',
    });
  });

  it('labels the service with its custom labels and other non-empty attributes', () => {
    expect(toManagedService(SERVICE_PAYLOAD).labels).toEqual({
      my_label: 'value',
      'app.kubernetes.io/name': 'db',
      environment: 'prod',
      cluster: 'cluster-1',
      version: '8.0.36',
      created_at: '2026-01-01T00:00:00Z',
    });
  });

  it('maps the main fields and the agents', () => {
    const service = toManagedService(SERVICE_PAYLOAD);

    expect(service).toMatchObject({
      serviceId: 'service-1',
      serviceType: ManagedServiceType.mysql,
      serviceName: 'mysql-1',
      nodeName: 'node-1',
      cluster: 'cluster-1',
      status: ServiceStatus.up,
      port: 3306,
    });
    expect(service.agents).toEqual([
      {
        agentId: 'agent-1',
        agentType: AgentType.mysqldExporter,
        status: AgentStatus.running,
        isConnected: false,
        disabled: false,
        properties: {},
      },
    ]);
  });
});

describe('toManagedNode', () => {
  it('maps the node with its other non-empty attributes as properties', () => {
    expect(toManagedNode(NODE_PAYLOAD)).toEqual({
      nodeId: 'node-1',
      nodeType: NodeType.generic,
      nodeName: 'node-1',
      address: '10.0.0.1',
      status: ServiceStatus.up,
      customLabels: { rack_id: '7' },
      isPmmServerNode: false,
      agents: [
        {
          agentId: 'pmm-agent-1',
          agentType: AgentType.pmmAgent,
          status: AgentStatus.running,
          isConnected: true,
          disabled: false,
          properties: {},
        },
      ],
      services: [
        {
          serviceId: 'service-1',
          serviceType: ManagedServiceType.mysql,
          serviceName: 'mysql-1',
        },
      ],
      properties: {
        machine_id: 'machine-1',
        is_pmm_server_node: 'false',
      },
    });
  });
});

describe('toManagedAgent', () => {
  it('flattens the parameters into properties', () => {
    expect(
      toManagedAgent({
        agent_id: 'agent-1',
        agent_type: AgentType.rdsExporter,
        status: AgentStatus.running,
        username: 'john',
        mongo_db_options: { mongo_opt_1: true, mongo_opt_2: ['foo', 'bar'] },
        rta_options: { collect_interval: '1s' },
        custom_labels: { additional_prop: 'prop_value' },
      }).properties
    ).toEqual({
      additional_prop: 'prop_value',
      username: 'john',
      mongo_opt_1: 'true',
      mongo_opt_2: 'foo,bar',
      'rta_options.collect_interval': '1s',
    });
  });

  it('skips empty parameters', () => {
    expect(
      toManagedAgent({
        agent_id: 'agent-1',
        agent_type: AgentType.nodeExporter,
        tls: false,
        listen_port: 0,
        metrics_path: '',
        azure_options: null,
      }).properties
    ).toEqual({});
  });

  it.each([
    [true, AgentStatus.running],
    [false, AgentStatus.unknown],
  ])(
    'derives a missing status from the connection (connected: %s)',
    (isConnected, status) => {
      expect(
        toManagedAgent({
          agent_id: 'agent-1',
          agent_type: AgentType.pmmAgent,
          status: '',
          is_connected: isConnected,
        }).status
      ).toBe(status);
    }
  );
});

describe('findRdsExporter', () => {
  it("picks the rds_exporter out of the node's agents", () => {
    expect(
      findRdsExporter([
        { agent_id: 'pmm-agent-1', agent_type: AgentType.pmmAgent },
        {
          agent_id: 'rds-1',
          agent_type: AgentType.rdsExporter,
          aws_access_key: '',
          aws_role_arn: 'arn:aws:iam::123456789012:role/Pmm',
          is_aws_secret_key_set: false,
        },
      ])
    ).toEqual({
      agentId: 'rds-1',
      awsAccessKey: '',
      awsRoleArn: 'arn:aws:iam::123456789012:role/Pmm',
      isAwsSecretKeySet: false,
    });
  });

  it('returns null when the node has no rds_exporter', () => {
    expect(
      findRdsExporter([
        { agent_id: 'pmm-agent-1', agent_type: AgentType.pmmAgent },
      ])
    ).toBeNull();
  });

  it('returns null when the node has more than one rds_exporter', () => {
    expect(
      findRdsExporter([
        { agent_id: 'rds-1', agent_type: AgentType.rdsExporter },
        { agent_id: 'rds-2', agent_type: AgentType.rdsExporter },
      ])
    ).toBeNull();
  });
});

describe('toBaseService', () => {
  it('maps the fields an inventory service carries', () => {
    expect(
      toBaseService({
        service_id: 'service-1',
        service_name: 'mysql-1',
        node_id: 'node-1',
        environment: 'prod',
        cluster: 'cluster-1',
        replication_set: 'rs-1',
        custom_labels: { my_label: 'value' },
      })
    ).toEqual({
      serviceId: 'service-1',
      serviceName: 'mysql-1',
      nodeId: 'node-1',
      environment: 'prod',
      cluster: 'cluster-1',
      replicationSet: 'rs-1',
      customLabels: { my_label: 'value' },
    });
  });
});

describe('getIfExists', () => {
  const originalAdapter = api.defaults.adapter;

  const stubApi = (status: number, data: object) => {
    api.defaults.adapter = (config: InternalAxiosRequestConfig) => {
      const response = {
        data: JSON.stringify(data),
        status,
        statusText: '',
        headers: {},
        config,
      } as AxiosResponse;

      return status >= 400
        ? Promise.reject(
            new AxiosError('', String(status), config, {}, response)
          )
        : Promise.resolve(response);
    };
  };

  afterEach(() => {
    api.defaults.adapter = originalAdapter;
    enqueueSnackbar.mockClear();
  });

  it('keeps custom label keys as the API returns them', async () => {
    stubApi(200, { node: { custom_labels: { my_label: 'value' } } });

    expect(await getIfExists('/management/nodes/node-1')).toEqual({
      node: { custom_labels: { my_label: 'value' } },
    });
  });

  it('resolves to null without a notification when the entity is gone', async () => {
    stubApi(404, { message: 'Node not found.' });

    expect(await getIfExists('/management/nodes/node-1')).toBeNull();
    expect(enqueueSnackbar).not.toHaveBeenCalled();
  });

  it('rejects and notifies on other errors', async () => {
    stubApi(500, { message: 'Internal error.' });

    await expect(getIfExists('/management/nodes/node-1')).rejects.toThrow();
    expect(enqueueSnackbar).toHaveBeenCalledWith(
      'Internal error.',
      expect.objectContaining({ variant: 'error' })
    );
  });
});
