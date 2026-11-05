import { useAddService } from 'hooks/api/useServices';
import type { FC } from 'react';
import { Messages as PageMessages } from '../../../AddServicePage.messages';
import { AddServiceType } from '../../../AddServicePage.types';
import { AdditionalOptions } from '../../../components/additional-options';
import { ConnectionDetails } from '../../../components/connection-details';
import { Labels } from '../../../components/labels';
import { MySQLOptions } from '../../mysql/mysql-options';
import { ServiceForm } from '../../ServiceForm';
import { useServiceForm } from '../../ServiceForm.hooks';
import { InstanceIdField } from '../instance-id-field';
import { RdsOptions } from '../rds-options';
import type { RdsServiceFormProps } from '../RdsServiceForm.types';
import { rdsMysqlSchema } from './RdsMySQLForm.schema';
import { toRdsMySQLDefaults, toRdsMySQLPayload } from './RdsMySQLForm.utils';

export const RdsMySQLForm: FC<RdsServiceFormProps> = ({
  instance,
  credentials,
  onCancel,
}) => {
  const form = useServiceForm(rdsMysqlSchema, toRdsMySQLDefaults(instance));
  const { mutateAsync: addService } = useAddService();

  return (
    <ServiceForm
      form={form}
      serviceTypeLabel={PageMessages.serviceTypes.mysql}
      onSubmit={(values) =>
        addService(toRdsMySQLPayload(values, instance, credentials))
      }
      onCancel={onCancel}
    >
      <ConnectionDetails requiresCredentials withNodes isAddressDisabled>
        <InstanceIdField />
      </ConnectionDetails>
      <Labels />
      <AdditionalOptions>
        <MySQLOptions cloud={AddServiceType.rds} />
        <RdsOptions />
      </AdditionalOptions>
    </ServiceForm>
  );
};
