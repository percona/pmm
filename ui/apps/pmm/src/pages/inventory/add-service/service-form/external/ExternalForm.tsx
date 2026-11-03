import { RadioGroup } from '@percona/peak-ui';
import { useAddService } from 'hooks/api/useServices';
import type { FC } from 'react';
import { useWatch } from 'react-hook-form';
import { Messages as PageMessages } from '../../AddServicePage.messages';
import { AdditionalOptions } from '../../components/additional-options';
import { Messages as ConnectionMessages } from '../../components/connection-details/ConnectionDetails.messages';
import { FieldGrid } from '../../components/field-grid';
import { PasswordField, TextField } from '../../components/fields';
import { FormSection } from '../../components/form-section';
import { Labels } from '../../components/labels';
import { TlsSkipVerify } from '../../components/tls-skip-verify';
import { ServiceForm } from '../ServiceForm';
import { useServiceForm } from '../ServiceForm.hooks';
import {
  EXTERNAL_DEFAULTS,
  METRICS_PARAMETERS_OPTIONS,
  SCHEME_OPTIONS,
} from './ExternalForm.constants';
import { Messages } from './ExternalForm.messages';
import { externalSchema } from './ExternalForm.schema';
import { MetricsParameters } from './ExternalForm.types';
import { toExternalPayload } from './ExternalForm.utils';

const { labels, placeholders, tooltips } = Messages;

export const ExternalForm: FC = () => {
  const form = useServiceForm(externalSchema, EXTERNAL_DEFAULTS);
  const metricsParameters = useWatch({
    control: form.control,
    name: 'metricsParameters',
  });
  const { mutateAsync: addService } = useAddService();

  return (
    <ServiceForm
      form={form}
      serviceTypeLabel={PageMessages.serviceTypes.external}
      onSubmit={(values) => addService(toExternalPayload(values))}
    >
      <FormSection title={Messages.title}>
        <FieldGrid>
          <TextField
            name="serviceName"
            label={ConnectionMessages.labels.serviceName}
            tooltip={ConnectionMessages.tooltips.serviceName}
            placeholder={ConnectionMessages.placeholders.serviceName}
          />
          <TextField
            name="group"
            label={labels.group}
            tooltip={tooltips.group}
          />
        </FieldGrid>
        <RadioGroup
          name="metricsParameters"
          label={labels.metricsParameters}
          options={METRICS_PARAMETERS_OPTIONS}
        />
        {metricsParameters === MetricsParameters.parsed ? (
          <TextField
            name="url"
            label={labels.url}
            tooltip={tooltips.url}
            placeholder={placeholders.url}
          />
        ) : (
          <>
            <RadioGroup
              name="scheme"
              label={labels.scheme}
              options={SCHEME_OPTIONS}
            />
            <FieldGrid>
              <TextField
                name="address"
                label={labels.address}
                tooltip={ConnectionMessages.tooltips.address}
              />
              <TextField
                name="metricsPath"
                label={labels.metricsPath}
                tooltip={tooltips.metricsPath}
                placeholder={placeholders.metricsPath}
              />
              <TextField
                name="port"
                label={labels.port}
                tooltip={tooltips.port}
              />
              <TextField
                name="username"
                label={labels.username}
                tooltip={tooltips.username}
              />
              <PasswordField
                name="password"
                label={labels.password}
                tooltip={tooltips.password}
              />
            </FieldGrid>
          </>
        )}
      </FormSection>
      <Labels />
      <AdditionalOptions>
        <TlsSkipVerify />
      </AdditionalOptions>
    </ServiceForm>
  );
};
