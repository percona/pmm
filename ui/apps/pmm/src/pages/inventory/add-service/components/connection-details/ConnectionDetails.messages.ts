export const Messages = {
  title: 'Main details',
  labels: {
    serviceName: 'Service name',
    address: 'Hostname',
    port: 'Port',
    username: 'Username',
    password: 'Password',
    timeout: 'Connection timeout',
  },
  placeholders: {
    serviceName: 'Service name (default: Hostname)',
    timeout: (defaultTimeout: string) => defaultTimeout,
  },
  tooltips: {
    serviceName: 'Service name to use',
    address: 'Public DNS hostname of your instance',
    port: 'Port your service is listening on',
    username: 'Your database user name',
    password: 'Your database password',
    timeout: (defaultTimeout: string) =>
      `How long PMM should wait when connecting to this service. Increase this for remote or high-latency databases. If the connection times out, PMM retries the next time it collects metrics. Leave empty to use the default of ${defaultTimeout}.`,
  },
};
