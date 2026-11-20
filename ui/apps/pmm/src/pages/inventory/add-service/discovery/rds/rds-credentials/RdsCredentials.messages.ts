export const Messages = {
  authMode: {
    label: 'Discover with',
    description:
      "An access key, an IAM role that PMM Server assumes with its own AWS identity, or PMM Server's own identity. These are mutually exclusive.",
    accessKey: 'Access key',
    iamRole: 'IAM role',
    serverIdentity: "PMM Server's identity",
  },
  accessKey: 'Amazon RDS access key',
  secretKey: 'Amazon RDS secret key',
  roleArn: {
    label: 'AWS IAM role ARN',
    placeholder: 'arn:aws:iam::123456789012:role/PmmRdsMonitoring',
    tooltip:
      'PMM Server assumes this role using its own AWS identity to discover RDS instances.',
  },
  serverIdentity:
    'PMM Server uses whatever AWS identity it already has, such as an instance profile or environment variables.',
  required: 'Required field',
  invalidRoleArn:
    'Enter an IAM role ARN, for example arn:aws:iam::123456789012:role/RoleName',
};
