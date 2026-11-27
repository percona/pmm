import { useAddService } from 'hooks/api/useServices';
import type { FC } from 'react';
import { Messages as PageMessages } from '../../AddServicePage.messages';
import { AdditionalOptions } from '../../components/additional-options';
import { ConnectionDetails } from '../../components/connection-details';
import { Labels } from '../../components/labels';
import { TlsCertificates } from '../../components/tls-certificates';
import { TlsOptions } from '../../components/tls-options';
import { ServiceForm } from '../ServiceForm';
import { useServiceForm } from '../ServiceForm.hooks';
import { VALKEY_DEFAULTS } from './ValkeyForm.constants';
import { valkeySchema } from './ValkeyForm.schema';
import { toValkeyPayload } from './ValkeyForm.utils';

export const ValkeyForm: FC = () => {
  const form = useServiceForm(valkeySchema, VALKEY_DEFAULTS);
  const { mutateAsync: addService } = useAddService();

  return (
    <ServiceForm
      form={form}
      serviceTypeLabel={PageMessages.serviceTypes.valkey}
      onSubmit={(values) => addService(toValkeyPayload(values))}
    >
      <ConnectionDetails withNodes defaultTimeout="3s" />
      <Labels />
      <AdditionalOptions>
        <TlsOptions certificates={<TlsCertificates />} />
      </AdditionalOptions>
    </ServiceForm>
  );
};
