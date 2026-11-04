export const Messages = {
  accessKey: 'Amazon RDS access key',
  secretKey: 'Amazon RDS secret key',
  roleArn: {
    label: 'AWS IAM role ARN',
    placeholder: 'arn:aws:iam::123456789012:role/PmmRdsMonitoring',
    tooltip:
      'PMM Server assumes this role using its own AWS identity to discover RDS instances. Use it instead of an access key and secret key, not together with them.',
  },
  invalidRoleArn:
    'Enter an IAM role ARN, for example arn:aws:iam::123456789012:role/RoleName',
  keysOrRole:
    'Use either an access key and secret key, or an IAM role ARN, not both',
};
