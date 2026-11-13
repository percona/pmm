import { RadioGroup } from '@percona/peak-ui';
import { useAddService } from 'hooks/api/useServices';
import { type FC, useEffect, useRef } from 'react';
import { useWatch } from 'react-hook-form';
import { AdditionalOptions } from '../../components/additional-options';
import { Messages as ConnectionMessages } from '../../components/connection-details/ConnectionDetails.messages';
import { FieldGrid } from '../../components/field-grid';
import { InfoTooltip, PasswordField, TextField } from '../../components/fields';
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
import { MetricsParameters, Scheme } from './ExternalForm.types';
import {
  getEndpointScheme,
  toExternalPayload,
  toManualFields,
  toSchemePort,
} from './ExternalForm.utils';

const { labels, placeholders, tooltips } = Messages;

export const ExternalForm: FC = () => {
  const form = useServiceForm(externalSchema, EXTERNAL_DEFAULTS);
  const [metricsParameters, url, scheme] = useWatch({
    control: form.control,
    name: ['metricsParameters', 'url', 'scheme'],
  });
  const isHttps =
    getEndpointScheme({ metricsParameters, url, scheme }) === Scheme.https;

  // Only a scheme the user picks moves the port; a parsed URL keeps the port it names.
  const selectScheme = (value: Scheme) => {
    const { formState, getValues, setValue } = form;
    setValue('scheme', value);
    setValue('port', toSchemePort(getValues('port'), value), {
      shouldValidate: formState.isSubmitted,
    });
  };

  const previousParameters = useRef(metricsParameters);
  useEffect(() => {
    const fields =
      previousParameters.current === MetricsParameters.parsed &&
      metricsParameters === MetricsParameters.manually &&
      toManualFields(form.getValues('url'));
    previousParameters.current = metricsParameters;

    if (fields) {
      const { setValue } = form;
      setValue('scheme', fields.scheme);
      setValue('address', fields.address);
      setValue('port', fields.port);
      setValue('metricsPath', fields.metricsPath);
      setValue('username', fields.username);
      setValue('password', fields.password);
    }
  }, [form, metricsParameters]);
  const { mutateAsync: addService } = useAddService();

  return (
    <ServiceForm
      form={form}
      serviceTypeLabel={Messages.serviceType}
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
          labelProps={{
            horizontalStackChildrenSlot: (
              <InfoTooltip title={tooltips.metricsParameters} />
            ),
          }}
          options={METRICS_PARAMETERS_OPTIONS}
        />
        {metricsParameters === MetricsParameters.parsed ? (
          <TextField
            name="url"
            label={labels.url}
            tooltip={tooltips.url}
            placeholder={placeholders.url}
            required
          />
        ) : (
          <>
            <RadioGroup
              name="scheme"
              label={labels.scheme}
              labelProps={{
                horizontalStackChildrenSlot: (
                  <InfoTooltip title={tooltips.scheme} />
                ),
              }}
              options={SCHEME_OPTIONS}
              // Replaces the form's own onChange, so the scheme is set here too.
              radioGroupFieldProps={{
                onChange: (_, value) => selectScheme(value as Scheme),
              }}
            />
            <FieldGrid>
              <TextField
                name="address"
                label={labels.address}
                tooltip={ConnectionMessages.tooltips.address}
                required
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
                required
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
      <AdditionalOptions>{isHttps && <TlsSkipVerify />}</AdditionalOptions>
    </ServiceForm>
  );
};
