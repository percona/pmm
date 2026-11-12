import { useAddService } from 'hooks/api/useServices';
import type { FC } from 'react';
import { Messages as PageMessages } from '../../../AddServicePage.messages';
import { AddServiceType } from '../../../AddServicePage.types';
import { AdditionalOptions } from '../../../components/additional-options';
import { ConnectionDetails } from '../../../components/connection-details';
import { Labels } from '../../../components/labels';
import { DatabaseField } from '../../postgresql/database-field';
import { PostgreSQLOptions } from '../../postgresql/postgresql-options';
import { ServiceForm } from '../../ServiceForm';
import { CLOUD_CONNECTION_TIMEOUT } from '../../ServiceForm.constants';
import { useServiceForm } from '../../ServiceForm.hooks';
import { InstanceIdField } from '../instance-id-field';
import { RdsOptions } from '../rds-options';
import type { RdsServiceFormProps } from '../RdsServiceForm.types';
import { rdsPostgresqlSchema } from './RdsPostgreSQLForm.schema';
import {
  toRdsPostgreSQLDefaults,
  toRdsPostgreSQLPayload,
} from './RdsPostgreSQLForm.utils';

export const RdsPostgreSQLForm: FC<RdsServiceFormProps> = ({
  instance,
  credentials,
  onCancel,
}) => {
  const form = useServiceForm(
    rdsPostgresqlSchema,
    toRdsPostgreSQLDefaults(instance)
  );
  const { mutateAsync: addService } = useAddService();

  return (
    <ServiceForm
      form={form}
      serviceTypeLabel={PageMessages.serviceTypes.postgresql}
      onSubmit={(values) =>
        addService(toRdsPostgreSQLPayload(values, instance, credentials))
      }
      onCancel={onCancel}
    >
      <ConnectionDetails
        requiresCredentials
        withNodes
        isAddressDisabled
        defaultTimeout={CLOUD_CONNECTION_TIMEOUT}
      >
        <InstanceIdField />
        <DatabaseField />
      </ConnectionDetails>
      <Labels />
      <AdditionalOptions>
        <PostgreSQLOptions cloud={AddServiceType.rds} />
        <RdsOptions />
      </AdditionalOptions>
    </ServiceForm>
  );
};
