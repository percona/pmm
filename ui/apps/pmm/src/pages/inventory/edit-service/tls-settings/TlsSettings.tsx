import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { SwitchInput, TextInput } from '@percona/peak-ui';
import type { FC } from 'react';
import type { ServiceAgent } from 'types/agent.types';
import type { ManagedServiceType } from 'types/services.types';
import { Messages } from '../EditService.messages';
import type { EditServiceFormValues } from '../EditService.schema';
import type { SettingName } from '../EditService.types';
import { SectionHeader } from '../section-header';
import { useIsSettingShown } from '../useIsSettingShown';

const { tls } = Messages;

const certificateFieldProps = (helperText: string) => ({
  multiline: true,
  minRows: 3,
  fullWidth: true,
  helperText,
});

interface Props {
  serviceType: ManagedServiceType;
  exporter: ServiceAgent;
}

export const TlsSettings: FC<Props> = ({ serviceType, exporter }) => {
  const isShown = useIsSettingShown(serviceType);
  const keyHelperText = exporter.isTlsKeySet ? tls.keyStored : tls.keepStored;
  const certificates: [SettingName, string, string][] = [
    ['tlsCa', tls.tlsCa, tls.keepStored],
    ['tlsCert', tls.tlsCert, tls.keepStored],
    ['tlsKey', tls.tlsKey, keyHelperText],
    ['tlsCertificateKey', tls.tlsCertificateKey, keyHelperText],
  ];
  const shownCertificates = certificates.filter(([name]) => isShown(name));

  return (
    <Stack gap={2} data-testid="tls-settings">
      <SectionHeader title={tls.title} />
      <Stack>
        <SwitchInput<EditServiceFormValues> name="tls" label={tls.tls} />
        <SwitchInput<EditServiceFormValues>
          name="tlsSkipVerify"
          label={tls.tlsSkipVerify}
        />
      </Stack>
      {!!shownCertificates.length && (
        <Typography variant="body2">{tls.certificatesDescription}</Typography>
      )}
      {shownCertificates.map(([name, label, helperText]) => (
        <TextInput<EditServiceFormValues>
          key={name}
          name={name}
          label={label}
          textFieldProps={certificateFieldProps(helperText)}
        />
      ))}
      {isShown('tlsCertificateKeyFilePassword') && (
        <TextInput<EditServiceFormValues>
          name="tlsCertificateKeyFilePassword"
          label={tls.tlsCertificateKeyFilePassword}
          textFieldProps={{
            type: 'password',
            autoComplete: 'new-password',
            fullWidth: true,
            helperText: exporter.isTlsCertificateKeyFilePasswordSet
              ? tls.keepStored
              : undefined,
          }}
        />
      )}
    </Stack>
  );
};
