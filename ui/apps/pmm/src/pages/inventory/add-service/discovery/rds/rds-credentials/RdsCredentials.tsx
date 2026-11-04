import type { FC } from 'react';
import { FieldGrid } from '../../../components/field-grid';
import { PasswordField, TextField } from '../../../components/fields';
import { useServiceForm } from '../../../service-form/ServiceForm.hooks';
import { toOptionalString } from '../../../service-form/ServiceForm.utils';
import { DiscoveryCredentials } from '../../components/discovery-credentials';
import { RDS_CREDENTIALS_DEFAULTS } from './RdsCredentials.constants';
import { Messages } from './RdsCredentials.messages';
import { rdsCredentialsSchema } from './RdsCredentials.schema';
import type {
  RdsCredentialsFormValues,
  RdsCredentialsProps,
} from './RdsCredentials.types';

export const RdsCredentials: FC<RdsCredentialsProps> = ({
  onDiscover,
  isDiscovering,
}) => {
  const form = useServiceForm(rdsCredentialsSchema, RDS_CREDENTIALS_DEFAULTS);

  const discover = (values: RdsCredentialsFormValues) =>
    onDiscover({
      awsAccessKey: toOptionalString(values.awsAccessKey),
      awsSecretKey: toOptionalString(values.awsSecretKey),
      awsRoleArn: toOptionalString(values.awsRoleArn),
    });

  return (
    <DiscoveryCredentials
      form={form}
      onSubmit={discover}
      isDiscovering={isDiscovering}
      testId="rds-credentials-form"
    >
      <FieldGrid>
        <TextField
          name="awsAccessKey"
          label={Messages.accessKey}
          placeholder={Messages.accessKey}
        />
        <PasswordField
          name="awsSecretKey"
          label={Messages.secretKey}
          placeholder={Messages.secretKey}
        />
      </FieldGrid>
      <TextField
        name="awsRoleArn"
        label={Messages.roleArn.label}
        placeholder={Messages.roleArn.placeholder}
        tooltip={Messages.roleArn.tooltip}
      />
    </DiscoveryCredentials>
  );
};
