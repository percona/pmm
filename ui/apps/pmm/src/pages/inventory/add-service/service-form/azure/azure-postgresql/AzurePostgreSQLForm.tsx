import { useAddAzureService } from 'hooks/api/useServices';
import type { FC } from 'react';
import { Messages as PageMessages } from '../../../AddServicePage.messages';
import { AddServiceType } from '../../../AddServicePage.types';
import { AdditionalOptions } from '../../../components/additional-options';
import { ConnectionDetails } from '../../../components/connection-details';
import { Labels } from '../../../components/labels';
import { PostgreSQLOptions } from '../../postgresql/postgresql-options';
import { ServiceForm } from '../../ServiceForm';
import { CLOUD_CONNECTION_TIMEOUT } from '../../ServiceForm.constants';
import { useServiceForm } from '../../ServiceForm.hooks';
import { AzureOptions } from '../azure-options';
import type { AzureServiceFormProps } from '../AzureServiceForm.types';
import { azurePostgresqlSchema } from './AzurePostgreSQLForm.schema';
import {
  toAzurePostgreSQLDefaults,
  toAzurePostgreSQLPayload,
} from './AzurePostgreSQLForm.utils';

export const AzurePostgreSQLForm: FC<AzureServiceFormProps> = ({
  instance,
  credentials,
  onCancel,
}) => {
  const form = useServiceForm(
    azurePostgresqlSchema,
    toAzurePostgreSQLDefaults(instance)
  );
  const { mutateAsync: addAzureService } = useAddAzureService();

  return (
    <ServiceForm
      form={form}
      serviceTypeLabel={PageMessages.serviceTypes.postgresql}
      onSubmit={(values) =>
        addAzureService(toAzurePostgreSQLPayload(values, instance, credentials))
      }
      onCancel={onCancel}
    >
      <ConnectionDetails
        requiresCredentials
        withNodes
        defaultTimeout={CLOUD_CONNECTION_TIMEOUT}
      />
      <Labels withCluster={false} />
      <AdditionalOptions>
        <PostgreSQLOptions cloud={AddServiceType.azure} />
        <AzureOptions />
      </AdditionalOptions>
    </ServiceForm>
  );
};
