export const Messages = {
  validation: {
    required: 'Required field',
    port: 'Port should be a number and between 0 and 65535',
    keyValue:
      'Values have to be in key:value format, and separated with new line or space',
    collectors:
      'Each collector name may contain only letters, numbers, "_", ".", "-", separated by commas',
    duration: 'Invalid duration',
    durationUnit: 'Invalid unit. Allowed units: s',
    minDuration: (min: string) =>
      `Duration should be greater or equal to ${min}`,
    maxDuration: (max: string) => `Duration should be lower or equal to ${max}`,
    min: (min: number) => `Value should be greater or equal to ${min}`,
    integer: 'Must be an Integer number',
  },
  success: (service: string, serviceType: string) =>
    `Service “${service}” added to your inventory. Your ${serviceType} service instance is now ready to be monitored.`,
  unknownError: 'Unknown error',
  addService: 'Add service',
};
