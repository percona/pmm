import { describe, expect, it } from 'vitest';
import { toServiceAgent } from 'api/inventory.utils';
import {
  AgentType,
  LogLevel,
  type RdsExporter,
  type ServiceAgent,
} from 'types/agent.types';
import {
  type EditableService,
  type InventoryService,
  ManagedServiceType,
  MetricsMode,
} from 'types/services.types';
import { MIB } from './EditService.constants';
import {
  type EditServiceFormValues,
  areCustomLabelsValid,
  areDsnParamsValid,
  getEditServiceSchema,
} from './EditService.schema';
import { QuerySource, RdsAuthMode } from './EditService.types';
import {
  getInitialValues,
  getRdsAuthMode,
  parseCustomLabels,
  toCustomLabelsText,
  toRdsCredentials,
  toServiceUpdate,
} from './EditService.utils';

const exporter = (overrides: Partial<RdsExporter> = {}): RdsExporter => ({
  agentId: 'rds-exporter-1',
  awsAccessKey: '',
  awsRoleArn: '',
  isAwsSecretKeySet: false,
  ...overrides,
});

const agent = (
  agentType: AgentType,
  overrides: Partial<ServiceAgent> = {}
): ServiceAgent => ({
  ...toServiceAgent({ agent_id: `${agentType}-1`, agent_type: agentType }),
  ...overrides,
});

const SERVICE: InventoryService = {
  serviceId: 'service-1',
  serviceType: ManagedServiceType.mysql,
  serviceName: 'mysql-1',
  nodeId: 'node-1',
  address: '127.0.0.1',
  port: 3306,
  socket: '',
  databaseName: '',
  group: '',
  environment: 'prod',
  cluster: 'cluster-1',
  replicationSet: 'rs-1',
  customLabels: { team: 'db', region_name: 'eu' },
};

const MYSQL_AGENTS = [
  agent(AgentType.mysqldExporter, {
    username: 'pmm',
    isPasswordSet: true,
    tls: true,
    logLevel: LogLevel.warn,
    connectionTimeout: '1.500s',
    disabledCollectors: ['heartbeat', 'info_schema.tables'],
    tablestatsGroupTableLimit: 1000,
  }),
  agent(AgentType.qanMysqlSlowlogAgent, {
    maxQueryLength: 2048,
    queryExamplesDisabled: true,
    commentsParsingDisabled: false,
    maxQueryLogSize: 512 * MIB,
  }),
];

const editable = (
  service: InventoryService = SERVICE,
  agents: ServiceAgent[] = MYSQL_AGENTS
): EditableService => ({ service, agents });

const INITIAL = getInitialValues(editable(), null);

const values = (
  overrides: Partial<EditServiceFormValues> = {}
): EditServiceFormValues => ({ ...INITIAL, ...overrides });

// The changes to the MySQL service for the given form values
const changes = (overrides: Partial<EditServiceFormValues>) =>
  toServiceUpdate(SERVICE, INITIAL, values(overrides))?.changes;

const ROLE_ARN = 'arn:aws:iam::123456789012:role/PmmRdsMonitoring';

describe('custom labels', () => {
  it('round-trips between the API map and the text', () => {
    expect(toCustomLabelsText(SERVICE.customLabels)).toBe(
      'team:db\nregion_name:eu'
    );
    expect(parseCustomLabels('team:db\n region_name:eu  ')).toEqual(
      SERVICE.customLabels
    );
  });

  it('keeps colons in label values', () => {
    const labels = { endpoint: 'db:3306', url: 'http://db' };

    expect(parseCustomLabels(toCustomLabelsText(labels))).toEqual(labels);
  });

  it('keeps spaces in label values', () => {
    const labels = { owner: 'DB Team', team: 'db' };

    expect(parseCustomLabels(toCustomLabelsText(labels))).toEqual(labels);
    expect(parseCustomLabels('owner:DB Team env:prod team:db')).toEqual({
      owner: 'DB Team',
      env: 'prod',
      team: 'db',
    });
  });

  it.each([
    ['', true],
    ['team:db\nregion_name:eu', true],
    ['team:db region_name:eu', true],
    ['team', false],
    ['team:', false],
    ['endpoint:db:3306', true],
    ['owner:DB Team', true],
    ['team:db 1team:db', false],
    ['1team:db', false],
    ['__reserved:db', false],
  ])('validates %j as %s', (text, valid) => {
    expect(areCustomLabelsValid(text)).toBe(valid);
  });
});

describe('getRdsAuthMode', () => {
  it('derives the mode from what the exporter holds', () => {
    expect(getRdsAuthMode(null)).toBe(RdsAuthMode.hostCredentials);
    expect(getRdsAuthMode(exporter())).toBe(RdsAuthMode.hostCredentials);
    expect(getRdsAuthMode(exporter({ awsRoleArn: ROLE_ARN }))).toBe(
      RdsAuthMode.iamRole
    );
    expect(getRdsAuthMode(exporter({ awsAccessKey: 'AKIA' }))).toBe(
      RdsAuthMode.accessKey
    );
  });

  it('treats a stored secret without access key as key authentication', () => {
    expect(getRdsAuthMode(exporter({ isAwsSecretKeySet: true }))).toBe(
      RdsAuthMode.accessKey
    );
  });
});

describe('toRdsCredentials', () => {
  it('does nothing without an exporter', () => {
    expect(
      toRdsCredentials(values({ rdsAuthMode: RdsAuthMode.iamRole }), null)
    ).toBeUndefined();
  });

  describe('migrating to an IAM role', () => {
    const keyExporter = exporter({
      awsAccessKey: 'AKIA',
      isAwsSecretKeySet: true,
    });

    it('clears both keys in the same call, trimming the ARN', () => {
      expect(
        toRdsCredentials(
          values({
            rdsAuthMode: RdsAuthMode.iamRole,
            awsRoleArn: ` ${ROLE_ARN} `,
          }),
          keyExporter
        )
      ).toEqual({ awsRoleArn: ROLE_ARN, awsAccessKey: '', awsSecretKey: '' });
    });

    it('skips the call when the role is unchanged', () => {
      expect(
        toRdsCredentials(
          values({ rdsAuthMode: RdsAuthMode.iamRole, awsRoleArn: ROLE_ARN }),
          exporter({ awsRoleArn: ROLE_ARN })
        )
      ).toBeUndefined();
    });
  });

  describe('migrating to an access key', () => {
    it('clears the role ARN in the same call', () => {
      expect(
        toRdsCredentials(
          values({
            rdsAuthMode: RdsAuthMode.accessKey,
            awsAccessKey: 'AKIA',
            awsSecretKey: 'secret',
          }),
          exporter({ awsRoleArn: ROLE_ARN })
        )
      ).toEqual({
        awsAccessKey: 'AKIA',
        awsRoleArn: '',
        awsSecretKey: 'secret',
      });
    });

    it('omits an untouched secret key so the stored one is kept', () => {
      expect(
        toRdsCredentials(
          values({ rdsAuthMode: RdsAuthMode.accessKey, awsAccessKey: 'AKIA2' }),
          exporter({ awsAccessKey: 'AKIA', isAwsSecretKeySet: true })
        )
      ).toEqual({ awsAccessKey: 'AKIA2', awsRoleArn: '' });
    });

    it('skips the call when nothing about the key changed', () => {
      expect(
        toRdsCredentials(
          values({ rdsAuthMode: RdsAuthMode.accessKey, awsAccessKey: 'AKIA' }),
          exporter({ awsAccessKey: 'AKIA', isAwsSecretKeySet: true })
        )
      ).toBeUndefined();
    });

    it('sends a new secret for the same access key', () => {
      expect(
        toRdsCredentials(
          values({
            rdsAuthMode: RdsAuthMode.accessKey,
            awsAccessKey: 'AKIA',
            awsSecretKey: 'new-secret',
          }),
          exporter({ awsAccessKey: 'AKIA', isAwsSecretKeySet: true })
        )
      ).toEqual({
        awsAccessKey: 'AKIA',
        awsRoleArn: '',
        awsSecretKey: 'new-secret',
      });
    });
  });

  describe('falling back to host credentials', () => {
    const hostValues = values({ rdsAuthMode: RdsAuthMode.hostCredentials });

    it('clears everything PMM holds', () => {
      expect(
        toRdsCredentials(hostValues, exporter({ awsRoleArn: ROLE_ARN }))
      ).toEqual({ awsAccessKey: '', awsSecretKey: '', awsRoleArn: '' });
    });

    it('skips the call when the exporter already uses them', () => {
      expect(toRdsCredentials(hostValues, exporter())).toBeUndefined();
    });

    it('still clears a stored secret that has no access key beside it', () => {
      expect(
        toRdsCredentials(hostValues, exporter({ isAwsSecretKeySet: true }))
      ).toEqual({ awsAccessKey: '', awsSecretKey: '', awsRoleArn: '' });
    });
  });
});

describe('getInitialValues', () => {
  it('reads the shared settings from the exporter, leaving secrets blank', () => {
    expect(INITIAL).toMatchObject({
      environment: 'prod',
      customLabels: 'team:db\nregion_name:eu',
      username: 'pmm',
      password: '',
      agentPassword: '',
      tls: true,
      tlsCa: '',
      tlsKey: '',
      metricsMode: MetricsMode.pull,
      logLevel: LogLevel.warn,
      connectionTimeout: '1.5',
      disableCollectors: 'heartbeat, info_schema.tables',
      tablestatsGroupTableLimit: '1000',
    });
  });

  it('reads the query source and its settings from the QAN agent', () => {
    expect(INITIAL).toMatchObject({
      querySource: QuerySource.slowlog,
      maxQueryLength: '2048',
      disableQueryExamples: true,
      disableCommentsParsing: false,
      maxSlowlogFileSize: '512',
    });
  });

  it('offers what the server gives a new QAN agent when there is none', () => {
    expect(
      getInitialValues(
        editable(SERVICE, [agent(AgentType.mysqldExporter)]),
        null
      )
    ).toMatchObject({
      querySource: QuerySource.none,
      maxQueryLength: '0',
      disableQueryExamples: false,
      disableCommentsParsing: true,
      maxSlowlogFileSize: '1024',
    });
  });

  it('shows a stored zero slowlog size as rotation turned off', () => {
    expect(
      getInitialValues(
        editable(SERVICE, [
          agent(AgentType.mysqldExporter),
          agent(AgentType.qanMysqlSlowlogAgent, { maxQueryLogSize: 0 }),
        ]),
        null
      ).maxSlowlogFileSize
    ).toBe('-1');
  });

  it('reads the push mode and the external endpoint', () => {
    const service = {
      ...SERVICE,
      serviceType: ManagedServiceType.external,
      group: 'redis',
    };

    expect(
      getInitialValues(
        editable(service, [
          agent(AgentType.externalExporter, {
            pushMetrics: true,
            listenPort: 9121,
            metricsPath: '/metrics',
          }),
        ]),
        null
      )
    ).toMatchObject({
      group: 'redis',
      metricsMode: MetricsMode.push,
      scheme: 'http',
      listenPort: '9121',
      metricsPath: '/metrics',
    });
  });
});

describe('toServiceUpdate', () => {
  it('returns nothing when no setting changed', () => {
    expect(toServiceUpdate(SERVICE, INITIAL, values())).toBeUndefined();
  });

  it('leaves stored labels alone when only another field changed', () => {
    const service = { ...SERVICE, customLabels: { owner: 'DB Team:Lead' } };
    const initial = getInitialValues(editable(service), null);

    expect(
      toServiceUpdate(service, initial, { ...initial, environment: 'staging' })
    ).toEqual({
      serviceId: SERVICE.serviceId,
      serviceType: ManagedServiceType.mysql,
      changes: { environment: 'staging' },
    });
  });

  it('sends only the changed labels, clearing emptied ones', () => {
    expect(changes({ cluster: 'cluster-2', environment: '' })).toEqual({
      cluster: 'cluster-2',
      environment: '',
    });
  });

  it('sends the whole custom labels map when it changed', () => {
    expect(changes({ customLabels: 'team:db' })).toEqual({
      customLabels: { team: 'db' },
    });
  });

  it('sends a secret only once one is typed', () => {
    expect(changes({ username: 'monitor', password: '' })).toEqual({
      username: 'monitor',
    });
    expect(changes({ password: 'secret' })).toEqual({ password: 'secret' });
  });

  it('sends certificates only while TLS is on', () => {
    expect(changes({ tlsCa: 'CA' })).toEqual({ tlsCa: 'CA' });
    expect(changes({ tls: false, tlsCa: 'CA' })).toEqual({ tls: false });
  });

  it('switches the query source by setting every QAN agent flag', () => {
    expect(changes({ querySource: QuerySource.perfschema })).toEqual({
      qanMysqlPerfschema: true,
      qanMysqlSlowlog: false,
    });
    expect(changes({ querySource: QuerySource.none })).toEqual({
      qanMysqlPerfschema: false,
      qanMysqlSlowlog: false,
    });
  });

  it('sends QAN settings only while a query source is chosen', () => {
    expect(changes({ maxQueryLength: '-1' })).toEqual({ maxQueryLength: -1 });
    expect(
      changes({ maxQueryLength: '-1', querySource: QuerySource.none })
    ).not.toHaveProperty('maxQueryLength');
    expect(
      changes({
        maxSlowlogFileSize: '256',
        querySource: QuerySource.perfschema,
      })
    ).not.toHaveProperty('maxSlowlogFileSize');
  });

  it('converts numbers, lists, durations and sizes', () => {
    expect(
      changes({
        tablestatsGroupTableLimit: '-1',
        disableCollectors: 'heartbeat, ,binlog_size,',
        connectionTimeout: '2.5',
        maxSlowlogFileSize: '256',
        metricsMode: MetricsMode.push,
        logLevel: LogLevel.debug,
      })
    ).toEqual({
      tablestatsGroupTableLimit: -1,
      disableCollectors: ['heartbeat', 'binlog_size'],
      connectionTimeout: '2.5s',
      maxSlowlogFileSize: 256 * MIB,
      metricsMode: MetricsMode.push,
      logLevel: LogLevel.debug,
    });
    expect(changes({ disableCollectors: '', connectionTimeout: '' })).toEqual({
      disableCollectors: [],
      connectionTimeout: '0s',
    });
    expect(changes({ maxSlowlogFileSize: '-1' })).toEqual({
      maxSlowlogFileSize: -1,
    });
  });

  it('parses the extra DSN parameters', () => {
    expect(changes({ extraDsnParams: 'tls:preferred\ncharset:utf8' })).toEqual({
      extraDsnParams: { tls: 'preferred', charset: 'utf8' },
    });
  });

  it('leaves out the settings the service type lacks', () => {
    const service = { ...SERVICE, serviceType: ManagedServiceType.proxysql };
    const initial = getInitialValues(
      editable(service, [agent(AgentType.proxysqlExporter)]),
      null
    );

    expect(
      toServiceUpdate(service, initial, {
        ...initial,
        disableQueryExamples: true,
        querySource: QuerySource.perfschema,
        tlsCa: 'CA',
        tls: true,
      })?.changes
    ).toEqual({ tls: true });
  });

  it('adds the leading slash to an external metrics path', () => {
    const service = { ...SERVICE, serviceType: ManagedServiceType.external };
    const initial = getInitialValues(
      editable(service, [agent(AgentType.externalExporter)]),
      null
    );

    expect(
      toServiceUpdate(service, initial, {
        ...initial,
        metricsPath: 'prometheus',
        listenPort: '9121',
        group: 'redis',
      })?.changes
    ).toEqual({ metricsPath: '/prometheus', listenPort: 9121, group: 'redis' });
  });
});

describe('getEditServiceSchema', () => {
  const errors = (
    formValues: EditServiceFormValues,
    rdsExporter: RdsExporter | null = null,
    serviceType = ManagedServiceType.mysql,
    initial = INITIAL
  ) => {
    const result = getEditServiceSchema(
      serviceType,
      initial,
      rdsExporter
    ).safeParse(formValues);

    return result.success
      ? {}
      : Object.fromEntries(
          result.error.issues.map(({ path, message }) => [path[0], message])
        );
  };

  it('checks the custom labels only once they are edited', () => {
    const initial = values({ customLabels: 'note:see y:' });

    expect(errors(initial, null, ManagedServiceType.mysql, initial)).toEqual(
      {}
    );
    expect(
      errors(
        values({ customLabels: 'note:see y: team:db' }),
        null,
        ManagedServiceType.mysql,
        initial
      )
    ).toEqual({
      customLabels:
        'Values have to be in key:value format, and separated with new line or space',
    });
  });

  it('checks the numbers only once they are edited', () => {
    const initial = values({ tablestatsGroupTableLimit: '-5' });

    expect(errors(initial, null, ManagedServiceType.mysql, initial)).toEqual(
      {}
    );
    expect(
      errors(values({ tablestatsGroupTableLimit: '-5', maxQueryLength: '1.5' }))
    ).toEqual({
      tablestatsGroupTableLimit: 'Enter a whole number of -1 or more',
      maxQueryLength: 'Enter a whole number of -1 or more',
    });
  });

  it('ignores the fields the form hides', () => {
    expect(
      errors(values({ querySource: QuerySource.none, maxQueryLength: 'x' }))
    ).toEqual({});
  });

  it('requires a username where the API does', () => {
    expect(errors(values({ username: ' ' }))).toEqual({
      username: 'Required field',
    });

    const service = { ...SERVICE, serviceType: ManagedServiceType.mongodb };
    const initial = getInitialValues(
      editable(service, [agent(AgentType.mongodbExporter, { username: 'u' })]),
      null
    );

    expect(
      errors(
        { ...initial, username: '' },
        null,
        ManagedServiceType.mongodb,
        initial
      )
    ).toEqual({});
  });

  it('checks the listen port range', () => {
    const service = { ...SERVICE, serviceType: ManagedServiceType.haproxy };
    const initial = getInitialValues(
      editable(service, [
        agent(AgentType.externalExporter, { listenPort: 8404 }),
      ]),
      null
    );
    const portError = (listenPort: string) =>
      errors(
        { ...initial, listenPort },
        null,
        ManagedServiceType.haproxy,
        initial
      );

    expect(portError('9000')).toEqual({});
    expect(portError('70000')).toEqual({
      listenPort: 'Enter a port between 1 and 65535',
    });
    expect(portError('')).toEqual({
      listenPort: 'Enter a port between 1 and 65535',
    });
  });

  it('takes a blank connection timeout but not a negative one', () => {
    expect(errors(values({ connectionTimeout: '' }))).toEqual({});
    expect(errors(values({ connectionTimeout: '-1' }))).toEqual({
      connectionTimeout: 'Enter a number of seconds, or leave blank',
    });
  });

  it('checks the extra DSN parameters', () => {
    expect(areDsnParamsValid('tls:preferred\ncharset:utf8')).toBe(true);
    expect(areDsnParamsValid('tls')).toBe(false);
    expect(errors(values({ extraDsnParams: 'tls' }))).toEqual({
      extraDsnParams:
        'Parameters have to be in name:value format, one per line',
    });
  });

  it('requires a valid role ARN in IAM role mode', () => {
    expect(
      errors(
        values({ rdsAuthMode: RdsAuthMode.iamRole, awsRoleArn: 'arn:bad' }),
        exporter()
      )
    ).toEqual({
      awsRoleArn:
        'Enter an IAM role ARN, for example arn:aws:iam::123456789012:role/RoleName',
    });
    expect(
      errors(
        values({ rdsAuthMode: RdsAuthMode.iamRole, awsRoleArn: ROLE_ARN }),
        exporter()
      )
    ).toEqual({});
  });

  it('keeps the stored secret while the access key is untouched', () => {
    const keyExporter = exporter({
      awsAccessKey: 'AKIA',
      isAwsSecretKeySet: true,
    });

    expect(
      errors(
        values({ rdsAuthMode: RdsAuthMode.accessKey, awsAccessKey: 'AKIA' }),
        keyExporter
      )
    ).toEqual({});
    expect(
      errors(
        values({ rdsAuthMode: RdsAuthMode.accessKey, awsAccessKey: 'AKIA2' }),
        keyExporter
      )
    ).toEqual({
      awsSecretKey: 'Enter the secret key that goes with this access key',
    });
  });

  it('ignores the credentials fields of the other modes', () => {
    expect(
      errors(values({ rdsAuthMode: RdsAuthMode.hostCredentials }), exporter())
    ).toEqual({});
  });
});
