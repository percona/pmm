import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { RadioGroup } from '@percona/peak-ui';
import type { FC } from 'react';
import { useWatch } from 'react-hook-form';
import { RdsAuthMode } from '../../../../Inventory.types';
import { FieldGrid } from '../../../components/field-grid';
import { PasswordField, TextField } from '../../../components/fields';
import { useServiceForm } from '../../../service-form/ServiceForm.hooks';
import { DiscoveryCredentials } from '../../components/discovery-credentials';
import { Messages } from './RdsCredentials.messages';
import { rdsCredentialsSchema } from './RdsCredentials.schema';
import type {
  RdsCredentialsFormValues,
  RdsCredentialsProps,
} from './RdsCredentials.types';
import {
  toDiscoverRdsPayload,
  toRdsCredentialsValues,
} from './RdsCredentials.utils';
import { RDS_AUTH_MODE_OPTIONS } from './RdsCredentials.constants';

const { authMode } = Messages;

export const RdsCredentials: FC<RdsCredentialsProps> = ({
  credentials,
  onDiscover,
  isDiscovering,
}) => {
  const form = useServiceForm(
    rdsCredentialsSchema,
    toRdsCredentialsValues(credentials)
  );
  const mode = useWatch({ control: form.control, name: 'rdsAuthMode' });

  const discover = (values: RdsCredentialsFormValues) =>
    onDiscover(toDiscoverRdsPayload(values));

  return (
    <DiscoveryCredentials
      form={form}
      onSubmit={discover}
      isDiscovering={isDiscovering}
      testId="rds-credentials-form"
    >
      <Stack gap={0.5}>
        <RadioGroup<RdsCredentialsFormValues>
          name="rdsAuthMode"
          label={authMode.label}
          options={RDS_AUTH_MODE_OPTIONS}
          radioGroupFieldProps={{ row: true }}
        />
        <Typography variant="body2" color="text.secondary">
          {authMode.description}
        </Typography>
      </Stack>
      {mode === RdsAuthMode.accessKey && (
        <FieldGrid>
          <TextField
            name="awsAccessKey"
            label={Messages.accessKey}
            placeholder={Messages.accessKey}
            required
          />
          <PasswordField
            name="awsSecretKey"
            label={Messages.secretKey}
            placeholder={Messages.secretKey}
            required
          />
        </FieldGrid>
      )}
      {mode === RdsAuthMode.iamRole && (
        <TextField
          name="awsRoleArn"
          label={Messages.roleArn.label}
          placeholder={Messages.roleArn.placeholder}
          tooltip={Messages.roleArn.tooltip}
          required
        />
      )}
      {mode === RdsAuthMode.hostCredentials && (
        <Typography variant="body2" data-testid="rds-server-identity-note">
          {Messages.serverIdentity}
        </Typography>
      )}
    </DiscoveryCredentials>
  );
};
