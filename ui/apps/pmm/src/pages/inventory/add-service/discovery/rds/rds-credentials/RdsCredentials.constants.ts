import type { RdsCredentialsFormValues } from './RdsCredentials.types';

// The same rule the API applies to aws_role_arn.
export const ROLE_ARN_RE = /^arn:aws[a-zA-Z0-9-]*:iam::[0-9]{12}:role\/.+$/;

export const RDS_CREDENTIALS_DEFAULTS: RdsCredentialsFormValues = {
  awsAccessKey: '',
  awsSecretKey: '',
  awsRoleArn: '',
};
