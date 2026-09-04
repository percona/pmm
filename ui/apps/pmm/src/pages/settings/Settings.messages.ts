import { PERCONA_SUPPORT_CONTACT_URL } from 'lib/constants';

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
      "Connect this PMM instance to Percona's ServiceNow to enable features like Support Diagnostics. Once connected, all PMM users can work with your organization's support cases directly from PMM.",
    // Step 1 — obtaining the credentials. Both links go to the same shortlink:
    // requesting credentials and buying a subscription are the same Support
    // conversation, and the two labels say which one the reader wants.
    getCredentialsStep: '1. Get your credentials',
    getCredentialsBody:
      'Open a Percona Support ticket to request a ServiceNow API key and client token.',
    requestCredentials: 'Request ServiceNow credentials',
    requestCredentialsLink: PERCONA_SUPPORT_CONTACT_URL,
    subscriptionPrompt: 'No Percona Support subscription?',
    subscriptionLink: PERCONA_SUPPORT_CONTACT_URL,
    subscriptionLinkText: 'Get Percona Support',
    // Step 2 — entering them.
    connectStep: '2. Connect',
    connectBody: 'Enter the credentials from your Percona Support ticket.',
    endpointLabel: 'ServiceNow endpoint (optional)',
    endpointHelper:
      'Leave it empty to use the default endpoint https://percona.service-now.com',
    // Copy for the credentials this PMM version's delivery plan is known to
    // declare. A name not listed here is a SEP build the UI has no copy for, so
    // it falls back to the generic label and helper below.
    secretCopy: {
      sn_api_key: {
        label: 'ServiceNow API key',
        helper: "Authenticates PMM with Percona's ServiceNow.",
      },
      client_token: {
        label: 'Client token',
        helper: 'Identifies your PMM instance to ServiceNow.',
      },
    } as Record<string, { label: string; helper: string }>,
    secretHelper: (name: string) => `Sent to SEP as "${name}".`,
    revealSecret: 'Show value',
    hideSecret: 'Hide value',
    noSecrets:
      "This PMM version's delivery plan declares no credentials, so only the ServiceNow endpoint can be set here.",
    submit: 'Verify and connect',
    submitting: 'Connecting…',
    cancelRenew: 'Cancel',
    // The connection as it stands. Only the endpoint is shown: PMM stores no
    // author or timestamp for a saved setting, so the rest of the design's
    // detail row waits on a SEP endpoint that can answer it.
    connectedTitle: 'ServiceNow connected',
    endpointDetailLabel: 'Endpoint',
    defaultEndpoint: 'https://percona.service-now.com',
    renew: 'Renew credentials',
    saveSuccess: 'ServiceNow connection to PMM successfully established.',
    driftedWarning:
      'The saved credentials no longer match what this PMM version expects. Enter them again to reconnect.',
    unavailable:
      "This PMM version doesn't include diagnostics delivery, so there's nothing to connect.",
    retry: 'Try again',
    disconnect: 'Disconnect',
    disconnectTitle: 'Disconnect ServiceNow?',
    disconnectBody:
      'PMM will forget the stored endpoint and credentials, and Support Diagnostics will stop sending results until they are supplied again.',
    disconnectConfirm: 'Disconnect',
    disconnectCancel: 'Cancel',
    disconnectSuccess: 'ServiceNow connection removed',
    loading: 'Loading the ServiceNow connection…',
    validation: {
      invalidUrl: 'Enter a valid URL (e.g. https://example.service-now.com/)',
      required: 'Required field',
    },
    errors: {
      forbidden:
        "Your account isn't allowed to change this connection. A PMM administrator has to save it.",
      unauthenticated:
        "Your session isn't valid for this action anymore. Reload the page and try again.",
      unreachable:
        "Couldn't reach the Support Diagnostics service. The previous configuration is unchanged.",
      generic:
        "Couldn't save the connection. The previous configuration is unchanged.",
      loadFailed:
        "Couldn't load the ServiceNow connection. This is usually temporary — try again, or reload the page.",
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
