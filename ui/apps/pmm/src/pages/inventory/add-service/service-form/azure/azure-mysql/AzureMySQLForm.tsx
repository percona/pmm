import { useAddAzureService } from 'hooks/api/useServices';
import type { FC } from 'react';
import { Messages as PageMessages } from '../../../AddServicePage.messages';
import { AdditionalOptions } from '../../../components/additional-options';
import { ConnectionDetails } from '../../../components/connection-details';
import { Labels } from '../../../components/labels';
import { MaxQueryLengthField } from '../../../components/max-query-length';
import { MySQLOptions } from '../../mysql/mysql-options';
import { ServiceForm } from '../../ServiceForm';
import { useServiceForm } from '../../ServiceForm.hooks';
import { AzureOptions } from '../azure-options';
import type { AzureServiceFormProps } from '../AzureServiceForm.types';
import { azureMysqlSchema } from './AzureMySQLForm.schema';
import {
  toAzureMySQLDefaults,
  toAzureMySQLPayload,
} from './AzureMySQLForm.utils';

export const AzureMySQLForm: FC<AzureServiceFormProps> = ({
  instance,
  credentials,
  onCancel,
}) => {
  const form = useServiceForm(azureMysqlSchema, toAzureMySQLDefaults(instance));
  const { mutateAsync: addAzureService } = useAddAzureService();

  return (
    <ServiceForm
      form={form}
      serviceTypeLabel={PageMessages.serviceTypes.mysql}
      onSubmit={(values) =>
        addAzureService(toAzureMySQLPayload(values, instance, credentials))
      }
      onCancel={onCancel}
    >
      <ConnectionDetails requiresCredentials withNodes>
        <MaxQueryLengthField />
      </ConnectionDetails>
      <Labels />
      <AdditionalOptions>
        <MySQLOptions withCollectors={false} />
        <AzureOptions />
      </AdditionalOptions>
    </ServiceForm>
  );
};
