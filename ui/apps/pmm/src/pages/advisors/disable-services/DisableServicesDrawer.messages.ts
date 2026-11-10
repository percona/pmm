export const Messages = {
  title: (summary: string) => `Disable "${summary}" for services`,
  description:
    "Stop this check on the services where its findings don't apply. It keeps running on all other services, and you can turn it back on at any time.",
  disabledGlobally:
    'This check is disabled for all services. Turn on its Status switch to choose services here. The services listed stay disabled after that.',
  servicesLabel: 'Services',
  disable: 'Disable',
  currentlyDisabled: 'Disabled for services',
  noDisabledServices: 'This check is not disabled for any service.',
  removedService: (serviceId: string) => `Removed service ${serviceId}`,
  enable: 'Re-enable',
  close: 'Close',
  success: {
    disabled: (count: number) =>
      `Check disabled for ${count} service${count === 1 ? '' : 's'}`,
    enabled: (serviceName: string) => `Check re-enabled for "${serviceName}"`,
  },
};
