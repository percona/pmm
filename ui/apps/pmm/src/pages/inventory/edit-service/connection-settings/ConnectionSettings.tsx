import Stack from '@mui/material/Stack';
import { RadioGroup, SwitchInput, TextInput } from '@percona/peak-ui';
import type { FC } from 'react';
import type { ServiceAgent } from 'types/agent.types';
import type { ManagedServiceType } from 'types/services.types';
import { METRICS_SCHEMES } from '../EditService.constants';
import { Messages } from '../EditService.messages';
import type { EditServiceFormValues } from '../EditService.schema';
import { SectionHeader } from '../section-header';
import { useIsSettingShown } from '../useIsSettingShown';

const { connection, endpoint } = Messages;

const SCHEME_OPTIONS = METRICS_SCHEMES.map((scheme) => ({
  value: scheme,
  label: scheme,
}));

// The API never returns secrets, so a blank field keeps the stored one
const secretFieldProps = (helperText?: string) => ({
  type: 'password',
  autoComplete: 'new-password',
  fullWidth: true,
  helperText,
});

interface Props {
  serviceType: ManagedServiceType;
  exporter: ServiceAgent;
}

export const ConnectionSettings: FC<Props> = ({ serviceType, exporter }) => {
  const isShown = useIsSettingShown(serviceType);
  // External exporters are scraped over HTTP rather than connecting to a database
  const isEndpoint = isShown('listenPort');
  const messages = isEndpoint ? endpoint : connection;
  const keepPassword = exporter.isPasswordSet ? messages.keepStored : undefined;

  return (
    <Stack gap={2} data-testid="connection-settings">
      <SectionHeader
        title={messages.title}
        description={messages.description}
      />
      {isEndpoint && (
        <>
          <RadioGroup<EditServiceFormValues>
            name="scheme"
            label={endpoint.scheme}
            options={SCHEME_OPTIONS}
          />
          <Stack direction={{ xs: 'column', md: 'row' }} gap={2}>
            <TextInput<EditServiceFormValues>
              name="listenPort"
              label={endpoint.listenPort}
              textFieldProps={{ type: 'number', fullWidth: true }}
            />
            <TextInput<EditServiceFormValues>
              name="metricsPath"
              label={endpoint.metricsPath}
              textFieldProps={{
                placeholder: endpoint.metricsPathPlaceholder,
                fullWidth: true,
              }}
            />
          </Stack>
        </>
      )}
      {isShown('database') && (
        <TextInput<EditServiceFormValues>
          name="database"
          label={connection.database}
          textFieldProps={{ fullWidth: true }}
        />
      )}
      <Stack direction={{ xs: 'column', md: 'row' }} gap={2}>
        <TextInput<EditServiceFormValues>
          name="username"
          label={messages.username}
          textFieldProps={{ autoComplete: 'off', fullWidth: true }}
        />
        <TextInput<EditServiceFormValues>
          name="password"
          label={messages.password}
          textFieldProps={secretFieldProps(keepPassword)}
        />
        {isShown('agentPassword') && (
          <TextInput<EditServiceFormValues>
            name="agentPassword"
            label={connection.agentPassword}
            textFieldProps={secretFieldProps(
              exporter.isAgentPasswordSet
                ? `${connection.agentPasswordDescription}. ${connection.keepStored}`
                : connection.agentPasswordDescription
            )}
          />
        )}
      </Stack>
      {isShown('authenticationMechanism') && (
        <Stack direction={{ xs: 'column', md: 'row' }} gap={2}>
          <TextInput<EditServiceFormValues>
            name="authenticationMechanism"
            label={connection.authenticationMechanism}
            textFieldProps={{
              fullWidth: true,
              helperText: connection.authenticationMechanismDescription,
            }}
          />
          <TextInput<EditServiceFormValues>
            name="authenticationDatabase"
            label={connection.authenticationDatabase}
            textFieldProps={{ fullWidth: true }}
          />
        </Stack>
      )}
      {isShown('extraDsnParams') && (
        <TextInput<EditServiceFormValues>
          name="extraDsnParams"
          label={connection.extraDsnParams}
          textFieldProps={{
            multiline: true,
            minRows: 2,
            fullWidth: true,
            placeholder: connection.extraDsnParamsPlaceholder,
            helperText: connection.extraDsnParamsDescription,
          }}
        />
      )}
      {isEndpoint && (
        <SwitchInput<EditServiceFormValues>
          name="tlsSkipVerify"
          label={endpoint.tlsSkipVerify}
        />
      )}
    </Stack>
  );
};
