export const Messages = {
  title: (summary: string) => `Disable "${summary}" for services`,
  description:
    'The check keeps running on all other services of its type. You can re-enable it for a service at any time.',
  disabledGlobally:
    'This check is disabled globally, so it does not run on any service. Enable it globally to manage per-service settings; the list below will still apply.',
  servicesLabel: 'Services',
  disable: 'Disable',
  currentlyDisabled: 'Disabled for services',
  noDisabledServices: 'This check is not disabled for any service.',
  removedService: (serviceId: string) => `Removed service (${serviceId})`,
  enable: 'Re-enable',
  close: 'Close',
  success: {
    disabled: (count: number) =>
      `Check disabled for ${count} service${count === 1 ? '' : 's'}`,
    enabled: (serviceName: string) => `Check re-enabled for "${serviceName}"`,
  },
};
