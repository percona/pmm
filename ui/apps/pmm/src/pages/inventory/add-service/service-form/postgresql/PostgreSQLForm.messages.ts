export const Messages = {
  database: {
    label: 'Database',
    placeholder: 'Database (default: postgres)',
    tooltip: 'Database name',
  },
  tracking: {
    label: 'Stat tracking options',
    options: {
      none: "Don't track",
      pgStatements: 'PG Stat Statements',
      pgMonitor: 'PG Stat Monitor',
    },
  },
  autoDiscovery: {
    label: 'Auto-discovery',
    limit: 'Auto-discovery limit',
    tooltip:
      'Turn off auto-discovery when the total count of databases exceeds the limit.',
    options: {
      enabled: 'Enabled',
      disabled: 'Disabled',
      custom: 'Custom',
    },
  },
  maxConnections: {
    label: 'Maximum connection limit',
    limit: 'Maximum connection limit per PostgreSQL instance',
    tooltip:
      'The maximum connections the Exporter can open to each PostgreSQL instance for monitoring. Higher values might be needed for larger or busier instances, but avoid setting it too high as it can impact performance. Adjust based on your instance size and monitoring needs.',
    options: {
      enabled: 'Enabled',
      disabled: 'Disabled',
    },
  },
};
