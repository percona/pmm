import type { FC } from 'react';
import { FieldGrid } from '../../../components/field-grid';
import { PasswordField, TextField } from '../../../components/fields';
import { useServiceForm } from '../../../service-form/ServiceForm.hooks';
import { DiscoveryCredentials } from '../../components/discovery-credentials';
import { AZURE_CREDENTIALS_DEFAULTS } from './AzureCredentials.constants';
import { Messages } from './AzureCredentials.messages';
import { azureCredentialsSchema } from './AzureCredentials.schema';
import type { AzureCredentialsProps } from './AzureCredentials.types';

export const AzureCredentials: FC<AzureCredentialsProps> = ({
  credentials,
  onDiscover,
  isDiscovering,
}) => {
  const form = useServiceForm(
    azureCredentialsSchema,
    credentials ?? AZURE_CREDENTIALS_DEFAULTS
  );

  return (
    <DiscoveryCredentials
      form={form}
      onSubmit={onDiscover}
      isDiscovering={isDiscovering}
      testId="azure-credentials-form"
    >
      <FieldGrid>
        <TextField
          name="azureClientId"
          label={Messages.clientId}
          placeholder={Messages.clientId}
        />
        <PasswordField
          name="azureClientSecret"
          label={Messages.clientSecret}
          placeholder={Messages.clientSecret}
        />
        <TextField
          name="azureTenantId"
          label={Messages.tenantId}
          placeholder={Messages.tenantId}
        />
        <TextField
          name="azureSubscriptionId"
          label={Messages.subscriptionId}
          placeholder={Messages.subscriptionId}
        />
      </FieldGrid>
    </DiscoveryCredentials>
  );
};
