import { LogLevel } from 'types/agent.types';
import { ManagedServiceType } from 'types/services.types';
import { QuerySource } from './EditService.types';

const KEEP_STORED = 'Leave blank to keep the current one';

export const Messages = {
  loadingTitle: 'Edit service',
  title: (serviceName: string) => `Editing “${serviceName}” service`,
  notFound: 'This service no longer exists.',
  cancel: 'Cancel',
  saveChanges: 'Save changes',
  success: (serviceName: string) =>
    `Service “${serviceName}” was changed. It is now ready to continue monitoring.`,
  partial:
    'AWS credentials were changed, the other settings were not. Review the error above and save them again.',
  details: {
    title: 'Service',
    description:
      'These details cannot be changed here. To change them, remove the service and add it again.',
    type: 'Type',
    node: 'Node',
    address: 'Address',
    socket: 'Socket',
    types: {
      [ManagedServiceType.mysql]: 'MySQL',
      [ManagedServiceType.mongodb]: 'MongoDB',
      [ManagedServiceType.postgresql]: 'PostgreSQL',
      [ManagedServiceType.proxysql]: 'ProxySQL',
      [ManagedServiceType.haproxy]: 'HAProxy',
      [ManagedServiceType.valkey]: 'Valkey',
      [ManagedServiceType.external]: 'External',
    },
  },
  labels: {
    title: 'Labels',
    description:
      'Please choose some labels to help identify your services. Labels can be useful, for example, to help define groupings. Editing existing labels may affect your inventory and its data.',
    environment: 'Environment',
    cluster: 'Cluster',
    replicationSet: 'Replication set',
    group: 'Group',
    customLabels: 'Custom labels',
    customLabelsDescription:
      'Follow the format as exemplified below, one label per line.',
    customLabelsPlaceholder: 'key1:value1\nkey2:value2',
    customLabelsInvalid:
      'Values have to be in key:value format, and separated with new line or space',
  },
  connection: {
    title: 'Connection',
    description:
      'The credentials and options the agents use to connect to the service. Passwords are never shown.',
    database: 'Database',
    username: 'Username',
    password: 'Password',
    agentPassword: 'Agent password',
    agentPasswordDescription:
      'A custom password for the exporter’s /metrics endpoint',
    keepStored: KEEP_STORED,
    authenticationMechanism: 'Authentication mechanism',
    authenticationMechanismDescription:
      'Leave blank for the default, or use MONGODB-X509 for certificates',
    authenticationDatabase: 'Authentication database',
    extraDsnParams: 'Extra DSN parameters',
    extraDsnParamsDescription: 'One parameter per line.',
    extraDsnParamsPlaceholder: 'param1:value1\nparam2:value2',
  },
  endpoint: {
    title: 'Metrics endpoint',
    description:
      'Where PMM scrapes the metrics from. Passwords are never shown.',
    scheme: 'Scheme',
    metricsPath: 'Metrics path',
    metricsPathPlaceholder: '/metrics',
    listenPort: 'Listen port',
    username: 'HTTP basic auth username',
    password: 'HTTP basic auth password',
    keepStored: KEEP_STORED,
    tlsSkipVerify: 'Skip TLS certificate and hostname verification',
  },
  tls: {
    title: 'TLS',
    tls: 'Use TLS for database connections',
    tlsSkipVerify: 'Skip TLS certificate and hostname verification',
    certificatesDescription:
      'Paste the contents of the files. The current certificates are never shown, so leave a field blank to keep what is stored.',
    tlsCa: 'TLS CA certificate',
    tlsCert: 'TLS client certificate',
    tlsKey: 'TLS client key',
    tlsCertificateKey: 'TLS certificate and key',
    tlsCertificateKeyFilePassword: 'TLS certificate key password',
    keyStored: 'A key is stored. Leave blank to keep it.',
    keepStored: KEEP_STORED,
  },
  qan: {
    title: 'Query Analytics',
    description:
      'Where PMM reads the queries from. Changing the source replaces the QAN agent.',
    querySource: 'Query source',
    sources: {
      [QuerySource.none]: 'None',
      [QuerySource.perfschema]: 'Performance Schema',
      [QuerySource.slowlog]: 'Slow log',
      [QuerySource.profiler]: 'Profiler',
      [QuerySource.mongolog]: 'Log file',
      [QuerySource.pgstatements]: 'pg_stat_statements',
      [QuerySource.pgstatmonitor]: 'pg_stat_monitor',
    },
    internalPostgresql:
      'Query Analytics for PMM’s internal PostgreSQL is turned on and off in Settings.',
    cloudNode:
      'On an Amazon RDS or Azure node, only the source that adding the service sets up is available.',
    maxQueryLength: 'Max query length',
    maxQueryLengthDescription: '0 uses the default, -1 means no limit',
    disableQueryExamples: 'Disable query examples',
    disableCommentsParsing: 'Disable comments parsing',
    maxSlowlogFileSize: 'Slow log rotation size (MiB)',
    maxSlowlogFileSizeDescription:
      'The slow log file is rotated at this size. 0 uses the default (1 GiB), -1 disables rotation.',
  },
  metrics: {
    title: 'Metrics',
    metricsMode: 'Metrics mode',
    pull: 'Pull',
    push: 'Push',
    logLevel: 'Log level',
    logLevels: {
      [LogLevel.unspecified]: 'Default',
      [LogLevel.fatal]: 'Fatal',
      [LogLevel.error]: 'Error',
      [LogLevel.warn]: 'Warning',
      [LogLevel.info]: 'Info',
      [LogLevel.debug]: 'Debug',
    },
    exposeExporter: 'Expose the exporter on all host addresses',
    connectionTimeout: 'Connection timeout (seconds)',
    connectionTimeoutDescription: 'Leave blank for the default',
    disableCollectors: 'Disabled collectors',
    disableCollectorsDescription: 'Collector names, separated by commas',
    tablestatsGroupTableLimit: 'Tablestats limit',
    tablestatsGroupTableLimitDescription:
      'Table statistics are not collected above this number of tables. 0 uses the default, -1 always disables them.',
    enableAllCollectors: 'Enable all collectors',
    enableDiagnosticDataHistograms:
      'Collect histogram metrics from getDiagnosticData',
    statsCollections: 'Stats collections',
    statsCollectionsDescription:
      'Databases or collections for collstats and indexstats, separated by commas',
    collectionsLimit: 'Max collections limit',
    collectionsLimitDescription:
      'Collection stats are not collected above this number of collections. 0 means no limit, -1 uses the default.',
    autoDiscoveryLimit: 'Auto-discovery limit',
    autoDiscoveryLimitDescription:
      'Databases are not discovered above this number of databases. 0 uses the default, -1 always disables it.',
    maxExporterConnections: 'Max exporter connections',
    maxExporterConnectionsDescription:
      'The most connections the exporter opens to the database. 0 uses the default.',
  },
  validation: {
    required: 'Required field',
    invalidNumber: (min: number) => `Enter a whole number of ${min} or more`,
    invalidPort: 'Enter a port between 1 and 65535',
    invalidTimeout: 'Enter a number of seconds, or leave blank',
    dsnParamsInvalid:
      'Parameters have to be in name:value format, one per line',
  },
  rds: {
    title: 'Amazon RDS credentials',
    description:
      'Choose how the RDS exporter authenticates to AWS for this service. Saving restarts the exporter, so expect a short gap in CloudWatch metrics.',
    authMode: {
      label: 'Authenticate with',
      description:
        'An access key, an IAM role assumed from the AWS identity of the pmm-agent host, or whatever credentials that host already provides. These are mutually exclusive.',
      accessKey: 'Access key',
      iamRole: 'IAM role',
      hostCredentials: 'Host credentials',
    },
    awsAccessKey: 'Amazon RDS access key',
    awsSecretKey: 'Amazon RDS secret key',
    awsSecretKeyStored: 'Leave blank to keep the current secret key',
    awsRoleArn: 'AWS IAM role ARN',
    awsRoleArnPlaceholder: 'arn:aws:iam::123456789012:role/PmmRdsMonitoring',
    awsRoleArnDescription:
      'The pmm-agent host running this exporter assumes this role using its own AWS identity. PMM stores no access key for it.',
    hostCredentials:
      'PMM will store no AWS credentials for this service. The exporter uses whatever the pmm-agent host provides - an instance profile, environment variables, or a credentials file. Monitoring stops if that host has none.',
    required: 'Required field',
    secretKeyRequired: 'Enter the secret key that goes with this access key',
    invalidRoleArn:
      'Enter an IAM role ARN, for example arn:aws:iam::123456789012:role/RoleName',
  },
  confirm: {
    description:
      'Changing existing labels can affect other parts of PMM dependent on it, such as: alerting, data in dashboards, etc. Find more details about it in ',
    docsLink: 'our documentation',
    settingsDescription:
      'The agents of this service restart with the new settings, so expect a short gap in its metrics and Query Analytics data.',
    skipConnectionCheck: 'Skip the connection check',
    skipConnectionCheckDescription:
      'Save even if PMM cannot connect to the service with the new settings. This applies to this save only.',
    confirm: 'Confirm and save changes',
    cancel: 'Cancel',
    cluster: {
      title: 'Cluster label changed',
      description:
        'Changing the cluster label will remove all scheduled backups for the impacted service/cluster. Make sure to recreate your backups after finishing the cluster configuration. For more information, see ',
      docsLink: 'Editing Labels',
    },
    credentials: {
      title: 'AWS credentials changed',
      description:
        'The RDS exporter restarts to pick up the new credentials, so expect a short gap in CloudWatch metrics for this service.',
    },
    hostCredentials: {
      title: 'PMM will stop storing AWS credentials',
      description:
        'The exporter will fall back to whatever credentials the pmm-agent host provides. If that host has none, monitoring for this service stops.',
    },
  },
};
