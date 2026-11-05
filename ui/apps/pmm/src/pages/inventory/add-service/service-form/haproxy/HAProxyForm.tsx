import { useAddService } from 'hooks/api/useServices';
import type { FC } from 'react';
import { Messages as PageMessages } from '../../AddServicePage.messages';
import { AdditionalOptions } from '../../components/additional-options';
import { Messages as ConnectionMessages } from '../../components/connection-details/ConnectionDetails.messages';
import { FieldGrid } from '../../components/field-grid';
import { PasswordField, TextField } from '../../components/fields';
import { FormSection } from '../../components/form-section';
import { Labels } from '../../components/labels';
import { ServiceForm } from '../ServiceForm';
import { useServiceForm } from '../ServiceForm.hooks';
import { HAPROXY_DEFAULTS } from './HAProxyForm.constants';
import { Messages } from './HAProxyForm.messages';
import { haproxySchema } from './HAProxyForm.schema';
import { toHAProxyPayload } from './HAProxyForm.utils';

const { labels, placeholders, tooltips } = ConnectionMessages;

export const HAProxyForm: FC = () => {
  const form = useServiceForm(haproxySchema, HAPROXY_DEFAULTS);
  const { mutateAsync: addService } = useAddService();

  return (
    <ServiceForm
      form={form}
      serviceTypeLabel={PageMessages.serviceTypes.haproxy}
      onSubmit={(values) => addService(toHAProxyPayload(values))}
    >
      <FormSection title={ConnectionMessages.title}>
        <FieldGrid>
          <TextField
            name="serviceName"
            label={labels.serviceName}
            tooltip={tooltips.serviceName}
            placeholder={placeholders.serviceName}
          />
          <TextField
            name="address"
            label={labels.address}
            tooltip={tooltips.address}
            required
          />
          <TextField
            name="port"
            label={labels.port}
            tooltip={Messages.tooltips.port}
            required
          />
          <TextField
            name="username"
            label={labels.username}
            tooltip={Messages.tooltips.username}
          />
          <PasswordField
            name="password"
            label={labels.password}
            tooltip={Messages.tooltips.password}
          />
        </FieldGrid>
      </FormSection>
      <Labels />
      <AdditionalOptions />
    </ServiceForm>
  );
};
