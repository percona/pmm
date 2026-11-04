import { useAddAzureService } from 'hooks/api/useServices';
import type { FC } from 'react';
import { Messages as PageMessages } from '../../../AddServicePage.messages';
import { AdditionalOptions } from '../../../components/additional-options';
import { ConnectionDetails } from '../../../components/connection-details';
import { Labels } from '../../../components/labels';
import { MaxQueryLengthField } from '../../../components/max-query-length';
import { DatabaseField } from '../../postgresql/database-field';
import { DISCOVERED_TRACKING_OPTIONS } from '../../postgresql/PostgreSQLForm.constants';
import { PostgreSQLOptions } from '../../postgresql/postgresql-options';
import { ServiceForm } from '../../ServiceForm';
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
      <ConnectionDetails withNodes>
        <DatabaseField />
        <MaxQueryLengthField />
      </ConnectionDetails>
      <Labels />
      <AdditionalOptions>
        <PostgreSQLOptions
          trackingOptions={DISCOVERED_TRACKING_OPTIONS}
          withExporterOptions={false}
        />
        <AzureOptions />
      </AdditionalOptions>
    </ServiceForm>
  );
};
