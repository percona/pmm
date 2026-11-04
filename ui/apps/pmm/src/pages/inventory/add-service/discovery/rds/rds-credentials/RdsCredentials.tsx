import { type FC, useEffect } from 'react';
import { FieldGrid } from '../../../components/field-grid';
import { PasswordField, TextField } from '../../../components/fields';
import { useServiceForm } from '../../../service-form/ServiceForm.hooks';
import { toOptionalString } from '../../../service-form/ServiceForm.utils';
import { DiscoveryCredentials } from '../../components/discovery-credentials';
import { Messages } from './RdsCredentials.messages';
import { rdsCredentialsSchema } from './RdsCredentials.schema';
import type {
  RdsCredentialsFormValues,
  RdsCredentialsProps,
} from './RdsCredentials.types';
import { toRdsCredentialsValues } from './RdsCredentials.utils';

export const RdsCredentials: FC<RdsCredentialsProps> = ({
  credentials,
  onDiscover,
  isDiscovering,
}) => {
  const form = useServiceForm(
    rdsCredentialsSchema,
    toRdsCredentialsValues(credentials)
  );

  // The rules span fields, so after a submit every change checks them all again.
  useEffect(() => {
    const { unsubscribe } = form.watch((_, { type }) => {
      if (type === 'change' && form.formState.isSubmitted) {
        form.trigger();
      }
    });
    return unsubscribe;
  }, [form]);

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
