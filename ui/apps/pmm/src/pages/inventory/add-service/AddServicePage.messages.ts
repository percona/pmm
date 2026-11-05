export const Messages = {
  serviceTypes: {
    rds: 'Amazon RDS',
    mysql: 'MySQL',
    mongodb: 'MongoDB',
    postgresql: 'PostgreSQL',
    proxysql: 'ProxySQL',
    haproxy: 'HAProxy',
    valkey: 'Valkey',
    external: 'External Service',
    azure: 'Microsoft Azure MySQL or PostgreSQL',
  },
  back: {
    selection: 'Select service type',
    instances: 'Discovered instances',
  },
  titles: {
    selection: 'Select service type',
    external: 'Add external service',
    configuring: (serviceType: string) => `Configuring ${serviceType} service`,
  },
};
