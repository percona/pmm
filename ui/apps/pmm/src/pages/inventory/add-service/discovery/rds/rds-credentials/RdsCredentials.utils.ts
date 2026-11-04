import type { DiscoverRDSPayload } from 'types/services.types';
import type { RdsCredentialsFormValues } from './RdsCredentials.types';

export const toRdsCredentialsValues = (
  credentials: DiscoverRDSPayload = {}
): RdsCredentialsFormValues => ({
  awsAccessKey: credentials.awsAccessKey ?? '',
  awsSecretKey: credentials.awsSecretKey ?? '',
  awsRoleArn: credentials.awsRoleArn ?? '',
});
