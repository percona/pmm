export const Messages = {
  title: 'Settings',
  tabs: {
    ssh: 'SSH key',
    metrics: 'Metrics resolution',
    advanced: 'Advanced settings',
    otel: 'OTEL',
  },
  advanced: {
    validation: {
      required: 'Required field',
      retentionRange: (min: number, max: number) =>
        `Value should be in the range from ${min} to ${max}`,
      intervalMin: (min: number) => `Min ${min}`,
    },
    retentionLabel: 'Data retention',
    retentionTooltip:
      'How long PMM keeps collected data. Older data is automatically deleted.',
    retentionUnits: 'days',
    retentionLink: 'https://per.co.na/data_retention',
    telemetryLabel: 'Telemetry',
    telemetryLink: 'https://per.co.na/telemetry',
    telemetryDialogLink: 'What we collect',
    telemetryTooltip:
      'Sends anonymous usage statistics to help improve PMM. No personal or database content is collected.',
    telemetrySummaryTitle:
      'We gather and send the following information to Percona:',
    updatesLabel: 'Check for updates',
    updatesLink: 'https://per.co.na/updates',
    updatesTooltip:
      'Option to check new versions and ability to update PMM from UI.',
    advisorsLabel: 'Advisors',
    sttRareIntervalLabel: 'Rare',
    sttStandardIntervalLabel: 'Standard',
    sttFrequentIntervalLabel: 'Frequent',
    sttCheckIntervalTooltip:
      'How often Advisor checks run. Lower values catch issues faster but increase resource usage.',
    advisorsLink: 'https://per.co.na/advisors',
    advisorsTooltip:
      'Run automated checks to identify potential database performance and configuration issues.',
    azureDiscoverLabel: 'Microsoft Azure monitoring',
    azureDiscoverTooltip:
      'Option to enable/disable Microsoft Azure DB instances discovery and monitoring',
    azureDiscoverLink: 'https://per.co.na/azure_monitoring',
    accessControl: 'Access control',
    accessControlTooltip:
      'Restrict data visibility based on user roles and labels.',
    accessControlLink: 'https://per.co.na/roles_permissions',
    publicAddressLabel: 'Public address',
    publicAddressTooltip:
      'The address or hostname PMM Server will be accessible at.',
    publicAddressPlaceholder: 'https://...',
    publicAddressButton: 'Get from browser',
    alertingLabel: 'Percona Alerting',
    alertingTooltip: 'Option to enable/disable Percona Alerting features.',
    alertingLink: 'https://per.co.na/alerting',
    backupLabel: 'Backup Management',
    backupTooltip:
      'Enable scheduled and on-demand backups for supported databases.',
    backupLink: 'https://per.co.na/backup_management',
    enableInternalPgQanLabel: 'QAN for PMM Server',
    enableInternalPgQanTooltip:
      "Displays queries from PMM Server's internal PostgreSQL database in Query Analytics (QAN). Enable to troubleshoot PMM Server's database performance alongside your monitored instances.",
    enableInternalPgQanLink: 'https://per.co.na/qan-pmm-server',
    nativeQanLabel: 'Native Query Analytics UI',
    nativeQanTooltip:
      'Use the native PMM Query Analytics page instead of the Grafana QAN panel. Technical preview — Grafana QAN remains available via direct link.',
    nativeQanLink: 'https://per.co.na/pmm-feature-status',
    featureManagementLabel: 'Feature management',
    featureManagementDescription:
      'Enable or disable core PMM capabilities. Turning off unused features can help conserve system resources and simplify your navigation menu.',
    technicalPreviewLegend: 'Technical preview features',
    technicalPreviewDescription: 'These are technical preview features, ',
    technicalPreviewWarning: 'not recommended',
    technicalPreviewDescriptionSuffix:
      ' to be used in production environments. Read more about feature status',
    technicalPreviewLinkText: 'here.',
  },
  metrics: {
    label: 'Metrics resolution',
    link: 'https://per.co.na/metrics_resolution',
    options: {
      rare: 'Rare',
      standard: 'Standard',
      frequent: 'Frequent',
      custom: 'Custom',
    },
    intervals: {
      low: 'Low',
      medium: 'Medium',
      high: 'High',
    },
    tooltip:
      'How often PMM collects metrics, in seconds. Lower values provide more detail but use more resources.',
    validation: {
      required: 'Required',
      minMax: (min: number, max: number) => `Must be between ${min} and ${max}`,
    },
  },
  ssh: {
    label: 'SSH key',
    link: 'https://per.co.na/ssh_key',
    tooltip:
      'Paste your public SSH key (ssh-rsa format) to enable SSH access to PMM Server.',
    placeholder: 'ssh-rsa AAAA...',
    validation: {
      invalidFormat: 'Enter a valid SSH public key (e.g. ssh-rsa, ssh-ed25519)',
    },
  },
  service: {
    success: 'Settings updated',
  },
  tooltipLinkText: 'Read more',
  unauthorized: 'Insufficient access permissions.',
  applyChanges: 'Apply changes',
  applying: 'Applying...',
  otel: {
    server: {
      title: 'Server OTEL',
      description:
        'Enable the OTEL collector on PMM Server (OTLP receiver) and set ClickHouse retention for logs, traces, and metrics.',
      collectorEnabledLabel: 'OTEL collector enabled',
      logsRetentionLabel: 'Log retention (days)',
      tracesRetentionLabel: 'Trace retention (days)',
      metricsRetentionLabel: 'Metrics retention (days)',
      disabledWarning:
        'When disabled, PMM Server stops receiving OTLP logs from agents. Existing ClickHouse data is kept until TTL expires.',
    },
    presets: {
      title: 'Log parser presets',
      summary: (builtIn: number, custom: number) =>
        `${builtIn} built-in + ${custom} custom preset${custom === 1 ? '' : 's'}.`,
      rawNote: 'Use preset name raw for unparsed log lines (not stored as a DB row).',
      addButton: 'Add preset',
      loading: 'Loading presets…',
      loadError: 'Failed to load log parser presets.',
      builtIn: 'Built-in',
      custom: 'Custom',
      usedBy: (n: number) => `Used by ${n} collector${n === 1 ? '' : 's'}`,
      edit: 'Edit',
      delete: 'Delete',
      deleteBlocked: 'Remove this preset from all collectors before deleting',
      deleteConfirm: (name: string) => `Delete custom preset "${name}"?`,
      deleted: 'Preset deleted',
      saved: 'Preset saved',
      createTitle: 'Add log parser preset',
      editTitle: 'Edit log parser preset',
      nameLabel: 'Name',
      nameHelp: 'Letters, digits, underscore; must start with a letter (used in pmm-admin and log_sources).',
      descriptionLabel: 'Description',
      yamlLabel: 'Operator YAML',
      yamlHelp:
        'YAML array of OTEL filelog operator objects (must include type on each item). Quote regex values and put each field on its own line.',
      invalidName: 'Invalid preset name',
      cancel: 'Cancel',
      save: 'Save',
      saving: 'Saving…',
    },
    collectors: {
      title: 'Log collectors',
      description:
        'OTEL collector agents on monitored nodes tail log files using parser presets. Install with pmm-admin add otel on nodes that do not appear here.',
      loading: 'Loading collectors…',
      loadError: 'Failed to load OTEL collectors.',
      empty: 'No OTEL collector agents found. Run pmm-admin add otel on a node with pmm-agent.',
      agentMeta: (id: string, status: string, count: number) =>
        `Agent ${id} · ${status} · ${count} log source${count === 1 ? '' : 's'}`,
      configure: 'Configure log sources',
      configureTitle: (node: string) => `Log sources — ${node}`,
      configureHelp: 'Each path is tailed by the node OTEL collector using the selected parser preset.',
      pathLabel: 'Log file path',
      presetLabel: 'Parser preset',
      addSource: 'Add log source',
      noSources: 'No log sources configured.',
      cancel: 'Cancel',
      save: 'Save',
      saving: 'Saving…',
      saved: 'Log sources updated',
    },
  },
};
