import { useAddService } from 'hooks/api/useServices';
import type { FC } from 'react';
import { Messages as PageMessages } from '../../AddServicePage.messages';
import { AdditionalOptions } from '../../components/additional-options';
import { ConnectionDetails } from '../../components/connection-details';
import { DisableCollectorsField } from '../../components/disable-collectors';
import { Labels } from '../../components/labels';
import { TlsOptions } from '../../components/tls-options';
import { ServiceForm } from '../ServiceForm';
import { useServiceForm } from '../ServiceForm.hooks';
import { PROXYSQL_DEFAULTS } from './ProxySQLForm.constants';
import { proxysqlSchema } from './ProxySQLForm.schema';
import { toProxySQLPayload } from './ProxySQLForm.utils';

export const ProxySQLForm: FC = () => {
  const form = useServiceForm(proxysqlSchema, PROXYSQL_DEFAULTS);
  const { mutateAsync: addService } = useAddService();

  return (
    <ServiceForm
      form={form}
      serviceTypeLabel={PageMessages.serviceTypes.proxysql}
      onSubmit={(values) => addService(toProxySQLPayload(values))}
    >
      <ConnectionDetails requiresCredentials withNodes />
      <Labels />
      <AdditionalOptions>
        <TlsOptions />
        <DisableCollectorsField />
      </AdditionalOptions>
    </ServiceForm>
  );
};
