import type { DiscoverRDSPayload } from 'types/services.types';
import { RdsAuthMode } from '../../../../Inventory.types';
import type { RdsCredentialsFormValues } from './RdsCredentials.types';

const getRdsAuthMode = (credentials: DiscoverRDSPayload) => {
  if (credentials.awsRoleArn) {
    return RdsAuthMode.iamRole;
  }

  return credentials.awsAccessKey
    ? RdsAuthMode.accessKey
    : RdsAuthMode.hostCredentials;
};

export const toRdsCredentialsValues = (
  credentials?: DiscoverRDSPayload
): RdsCredentialsFormValues => ({
  rdsAuthMode: credentials
    ? getRdsAuthMode(credentials)
    : RdsAuthMode.accessKey,
  awsAccessKey: credentials?.awsAccessKey ?? '',
  awsSecretKey: credentials?.awsSecretKey ?? '',
  awsRoleArn: credentials?.awsRoleArn ?? '',
});

// Only the chosen mode's fields are sent, since the server rejects keys next to a role.
export const toDiscoverRdsPayload = ({
  rdsAuthMode,
  awsAccessKey,
  awsSecretKey,
  awsRoleArn,
}: RdsCredentialsFormValues): DiscoverRDSPayload => {
  if (rdsAuthMode === RdsAuthMode.accessKey) {
    return {
      awsAccessKey: awsAccessKey.trim(),
      awsSecretKey: awsSecretKey.trim(),
    };
  }

  return rdsAuthMode === RdsAuthMode.iamRole
    ? { awsRoleArn: awsRoleArn.trim() }
    : {};
};
