import { useAddService } from 'hooks/api/useServices';
import type { FC } from 'react';
import { Messages as PageMessages } from '../../AddServicePage.messages';
import { AdditionalOptions } from '../../components/additional-options';
import { ConnectionDetails } from '../../components/connection-details';
import { Labels } from '../../components/labels';
import { MaxQueryLengthField } from '../../components/max-query-length';
import { ServiceForm } from '../ServiceForm';
import { useServiceForm } from '../ServiceForm.hooks';
import { MYSQL_DEFAULTS } from './MySQLForm.constants';
import { mysqlSchema } from './MySQLForm.schema';
import { toMySQLPayload } from './MySQLForm.utils';
import { MySQLOptions } from './mysql-options';

export const MySQLForm: FC = () => {
  const form = useServiceForm(mysqlSchema, MYSQL_DEFAULTS);
  const { mutateAsync: addService } = useAddService();

  return (
    <ServiceForm
      form={form}
      serviceTypeLabel={PageMessages.serviceTypes.mysql}
      onSubmit={(values) => addService(toMySQLPayload(values))}
    >
      <ConnectionDetails withNodes>
        <MaxQueryLengthField />
      </ConnectionDetails>
      <Labels />
      <AdditionalOptions>
        <MySQLOptions />
      </AdditionalOptions>
    </ServiceForm>
  );
};
