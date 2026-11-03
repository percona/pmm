import { SwitchInput } from '@percona/peak-ui';
import { useAddService } from 'hooks/api/useServices';
import type { FC } from 'react';
import { Messages as PageMessages } from '../../AddServicePage.messages';
import { AdditionalOptions } from '../../components/additional-options';
import { ConnectionDetails } from '../../components/connection-details';
import { DisableCollectorsField } from '../../components/disable-collectors';
import { Labels } from '../../components/labels';
import { MaxQueryLengthField } from '../../components/max-query-length';
import { MongodbTlsCertificates } from '../../components/mongodb-tls-certificates';
import { TlsOptions } from '../../components/tls-options';
import { ServiceForm } from '../ServiceForm';
import { useServiceForm } from '../ServiceForm.hooks';
import { MONGODB_DEFAULTS } from './MongoDBForm.constants';
import { Messages } from './MongoDBForm.messages';
import { mongodbSchema } from './MongoDBForm.schema';
import { toMongoDBPayload } from './MongoDBForm.utils';

export const MongoDBForm: FC = () => {
  const form = useServiceForm(mongodbSchema, MONGODB_DEFAULTS);
  const { mutateAsync: addService } = useAddService();

  return (
    <ServiceForm
      form={form}
      serviceTypeLabel={PageMessages.serviceTypes.mongodb}
      onSubmit={(values) => addService(toMongoDBPayload(values))}
    >
      <ConnectionDetails withNodes>
        <MaxQueryLengthField />
      </ConnectionDetails>
      <Labels />
      <AdditionalOptions>
        <TlsOptions certificates={<MongodbTlsCertificates />} />
        <SwitchInput
          name="qanMongodbProfiler"
          label={Messages.qanMongodbProfiler}
        />
        <DisableCollectorsField />
      </AdditionalOptions>
    </ServiceForm>
  );
};
