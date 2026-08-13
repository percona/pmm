import { PERCONA_SUPPORT_URL } from './Settings.constants';

export const Messages = {
  title: 'Settings',
  tabs: {
    ssh: 'SSH key',
    metrics: 'Metrics resolution',
    advanced: 'Advanced settings',
    serviceNow: 'ServiceNow connection',
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
  serviceNow: {
    label: 'ServiceNow connection',
    description:
      "Connect this PMM instance to Percona's ServiceNow so features like Support diagnostics can work directly with your support cases. You'll need a Percona Support account with access to your organization's cases.",
    scopeNote:
      "This connection applies to the whole PMM instance. Once connected, anyone with access to Support features can send results to your organization's cases.",
    subscriptionPrompt: "Don't have a Percona Support subscription?",
    subscriptionLink: PERCONA_SUPPORT_URL,
    subscriptionLinkText: 'Learn about Percona Support',
    endpointLabel: 'Receiver endpoint',
    endpointPlaceholder: 'https://percona.service-now.com/',
    endpointHelper:
      'Leave empty to use the receiver bundled with this PMM version.',
    secretsLegend: 'Credentials',
    secretHelper: (name: string) => `Sent to SEP as "${name}".`,
    secretStoredHelper: (name: string) =>
      `Sent to SEP as "${name}". A value is stored — leave it untouched to keep it.`,
    status: {
      configured:
        'Connected. Support diagnostics can send results to your ServiceNow cases.',
      notConfigured:
        "Not configured. Support diagnostics can't send results until these details are saved.",
      drifted:
        "The stored details no longer match this deployment's delivery plan. Re-supply the values below.",
    },
    noSecrets:
      "This PMM version's delivery plan declares no credentials, so only the receiver endpoint can be set here.",
    unavailable:
      'Support diagnostics delivery is not part of this PMM version, so there is nothing to connect.',
    saveSuccess: 'ServiceNow connection saved',
    disconnect: 'Disconnect',
    disconnectTitle: 'Disconnect ServiceNow?',
    disconnectBody:
      'PMM will forget the stored endpoint and credentials, and Support diagnostics will stop sending results until they are supplied again.',
    disconnectConfirm: 'Disconnect',
    disconnectCancel: 'Cancel',
    disconnectSuccess: 'ServiceNow connection removed',
    validation: {
      invalidUrl: 'Enter a valid URL (e.g. https://example.service-now.com/)',
    },
    errors: {
      forbidden:
        "Your account isn't allowed to change this connection. A PMM administrator has to save it.",
      unauthenticated:
        "Your session isn't valid for this action anymore. Reload the page and try again.",
      unreachable:
        "Couldn't reach the Support diagnostics service. The previous configuration is unchanged.",
      generic:
        "Couldn't save the connection. The previous configuration is unchanged.",
      loadFailed: "Couldn't load the current ServiceNow connection.",
    },
  },
  service: {
    success: 'Settings updated',
  },
  tooltipLinkText: 'Read more',
  unauthorized: 'Insufficient access permissions.',
  applyChanges: 'Apply changes',
  applying: 'Applying...',
};
