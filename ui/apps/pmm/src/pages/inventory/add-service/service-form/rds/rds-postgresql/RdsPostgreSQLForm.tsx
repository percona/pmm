import { useAddService } from 'hooks/api/useServices';
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
      <ConnectionDetails requiresCredentials withNodes isAddressDisabled>
        <InstanceIdField />
        <DatabaseField />
        <MaxQueryLengthField />
      </ConnectionDetails>
      <Labels />
      <AdditionalOptions>
        <PostgreSQLOptions trackingOptions={DISCOVERED_TRACKING_OPTIONS} />
        <RdsOptions />
      </AdditionalOptions>
    </ServiceForm>
  );
};
