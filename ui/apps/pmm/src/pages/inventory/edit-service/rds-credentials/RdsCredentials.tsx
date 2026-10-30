import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { RadioGroup, TextInput } from '@percona/peak-ui';
import type { FC } from 'react';
import { useWatch } from 'react-hook-form';
import type { RdsExporter } from 'types/agent.types';
import { Messages } from '../EditService.messages';
import type { EditServiceFormValues } from '../EditService.schema';
import { RdsAuthMode } from '../EditService.types';

const { rds } = Messages;

const AUTH_MODE_OPTIONS = [
  { value: RdsAuthMode.accessKey, label: rds.authMode.accessKey },
  { value: RdsAuthMode.iamRole, label: rds.authMode.iamRole },
  { value: RdsAuthMode.hostCredentials, label: rds.authMode.hostCredentials },
];

interface Props {
  exporter: RdsExporter;
}

export const RdsCredentials: FC<Props> = ({ exporter }) => {
  const mode = useWatch<EditServiceFormValues, 'rdsAuthMode'>({
    name: 'rdsAuthMode',
  });

  return (
    <Stack gap={2} data-testid="rds-credentials">
      <Stack gap={0.5}>
        <Typography variant="h6">{rds.title}</Typography>
        <Typography variant="body2">{rds.description}</Typography>
      </Stack>
      <Stack gap={0.5}>
        <RadioGroup<EditServiceFormValues>
          name="rdsAuthMode"
          label={rds.authMode.label}
          options={AUTH_MODE_OPTIONS}
          radioGroupFieldProps={{ row: true }}
        />
        <Typography variant="body2" color="text.secondary">
          {rds.authMode.description}
        </Typography>
      </Stack>
      {mode === RdsAuthMode.accessKey && (
        <Stack direction={{ xs: 'column', md: 'row' }} gap={2}>
          <TextInput<EditServiceFormValues>
            name="awsAccessKey"
            label={rds.awsAccessKey}
            textFieldProps={{ placeholder: rds.awsAccessKey, fullWidth: true }}
          />
          <TextInput<EditServiceFormValues>
            name="awsSecretKey"
            label={rds.awsSecretKey}
            textFieldProps={{
              type: 'password',
              autoComplete: 'new-password',
              fullWidth: true,
              placeholder: exporter.isAwsSecretKeySet
                ? rds.awsSecretKeyStored
                : rds.awsSecretKey,
            }}
          />
        </Stack>
      )}
      {mode === RdsAuthMode.iamRole && (
        <TextInput<EditServiceFormValues>
          name="awsRoleArn"
          label={rds.awsRoleArn}
          textFieldProps={{
            placeholder: rds.awsRoleArnPlaceholder,
            helperText: rds.awsRoleArnDescription,
            fullWidth: true,
          }}
        />
      )}
      {mode === RdsAuthMode.hostCredentials && (
        <Typography variant="body2" data-testid="rds-host-credentials-note">
          {rds.hostCredentials}
        </Typography>
      )}
    </Stack>
  );
};
