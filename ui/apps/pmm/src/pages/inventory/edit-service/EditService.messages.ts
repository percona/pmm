export const Messages = {
  loadingTitle: 'Edit service',
  title: (serviceName: string) => `Editing “${serviceName}” service`,
  notFound: 'This service no longer exists.',
  cancel: 'Cancel',
  saveChanges: 'Save changes',
  success: (serviceName: string) =>
    `Service “${serviceName}” was changed. It is now ready to continue monitoring.`,
  partial:
    'AWS credentials were changed, labels were not. Review the error above and save the label changes again.',
  labels: {
    title: 'Labels',
    description:
      'Please choose some labels to help identify your services. Labels can be useful, for example, to help define groupings. Editing existing labels may affect your inventory and its data.',
    environment: 'Environment',
    cluster: 'Cluster',
    replicationSet: 'Replication set',
    customLabels: 'Custom labels',
    customLabelsDescription:
      'Follow the format as exemplified below, one label per line.',
    customLabelsPlaceholder: 'key1:value1\nkey2:value2',
    customLabelsInvalid:
      'Values have to be in key:value format, and separated with new line or space',
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
