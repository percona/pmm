import { useAddService } from 'hooks/api/useServices';
import type { FC } from 'react';
import { Messages as PageMessages } from '../../AddServicePage.messages';
import { AdditionalOptions } from '../../components/additional-options';
import { ConnectionDetails } from '../../components/connection-details';
import { Labels } from '../../components/labels';
import { MaxQueryLengthField } from '../../components/max-query-length';
import { ServiceForm } from '../ServiceForm';
import { useServiceForm } from '../ServiceForm.hooks';
import { DatabaseField } from './database-field';
import { POSTGRESQL_DEFAULTS } from './PostgreSQLForm.constants';
import { postgresqlSchema } from './PostgreSQLForm.schema';
import { toPostgreSQLPayload } from './PostgreSQLForm.utils';
import { PostgreSQLOptions } from './postgresql-options';

export const PostgreSQLForm: FC = () => {
  const form = useServiceForm(postgresqlSchema, POSTGRESQL_DEFAULTS);
  const { mutateAsync: addService } = useAddService();

  return (
    <ServiceForm
      form={form}
      serviceTypeLabel={PageMessages.serviceTypes.postgresql}
      onSubmit={(values) => addService(toPostgreSQLPayload(values))}
    >
      <ConnectionDetails requiresCredentials withNodes>
        <DatabaseField />
        <MaxQueryLengthField />
      </ConnectionDetails>
      <Labels />
      <AdditionalOptions>
        <PostgreSQLOptions />
      </AdditionalOptions>
    </ServiceForm>
  );
};
