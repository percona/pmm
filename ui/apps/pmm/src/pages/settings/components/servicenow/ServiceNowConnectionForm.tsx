import { FC, useMemo, useState } from 'react';
import Alert from '@mui/material/Alert';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import Divider from '@mui/material/Divider';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { formControlClasses } from '@mui/material';
import { TextInput } from '@percona/peak-ui';
import { zodResolver } from '@hookform/resolvers/zod';
import { FormProvider, useForm } from 'react-hook-form';
import { enqueueSnackbar } from 'notistack';
import {
  type ApiError,
  useResetSetting,
  usePatchSetting,
  useSettingsList,
} from '@sep/api';
import { Modal } from 'components/modal';
import { helperTextTestId } from 'utils/mui.utils';
import { Messages } from '../../Settings.messages';
import { MAX_LABEL_WIDTH } from '../../Settings.constants';
import { SettingsFieldLabel } from '../settings-field-label';
import { SettingsSubmitButton } from '../settings-submit-button';
import {
  DELIVERY_INPUTS_KEY,
  SEP_SETTINGS_CLASS,
} from './ServiceNowConnection.constants';
import { serviceNowSchema } from './ServiceNowConnectionForm.schema';
import { ServiceNowFormValues } from './ServiceNowConnection.types';
import {
  buildDeliveryInputsPatch,
  connectionStatus,
  declaredSecretNames,
  sepErrorMessage,
  secretLabel,
  storedDeliveryInputs,
  toFormValues,
} from './ServiceNowConnection.utils';

const STATUS_SEVERITY = {
  configured: 'success',
  'not-configured': 'info',
  drifted: 'warning',
} as const;

const STATUS_MESSAGE = {
  configured: Messages.serviceNow.status.configured,
  'not-configured': Messages.serviceNow.status.notConfigured,
  drifted: Messages.serviceNow.status.drifted,
} as const;

/**
 * Direct entry of the ServiceNow details SEP needs to deliver diagnostics.
 *
 * The operator obtains a ServiceNow token out of band and enters it here;
 * PMM-15218 replaces this entry surface with a guided round trip and keeps the
 * write path below untouched.
 *
 * The write is one whole-object PATCH of `DIAGNOSTICS_DELIVERY_INPUTS` carrying
 * exactly the secret names the SEP image declares — SEP seals the leaves and
 * rejects an unexpected name, so neither is a shape the UI may improvise. All
 * validation is server-side and all-or-nothing: a rejected save leaves the
 * previous configuration standing, which is why nothing here is optimistic.
 */
export const ServiceNowConnectionForm: FC = () => {
  const { data: groups, isLoading, error: loadError } = useSettingsList();
  const { mutateAsync: patchSetting, error: saveError } = usePatchSetting();
  const { mutateAsync: resetSetting, isPending: isDisconnecting } =
    useResetSetting();
  const [disconnectOpen, setDisconnectOpen] = useState(false);

  const declaredNames = useMemo(() => declaredSecretNames(groups), [groups]);
  const stored = useMemo(() => storedDeliveryInputs(groups), [groups]);
  const status = connectionStatus(declaredNames, stored);

  // `values` (not `defaultValues`) so a refetch — the invalidation after a save,
  // in particular — re-seeds the fields with what SEP actually stored. React
  // Hook Form only re-seeds on a deep change, so a background refetch that
  // returns the same data leaves half-typed input alone.
  const values = useMemo(
    () => toFormValues(declaredNames, stored),
    [declaredNames, stored]
  );
  const methods = useForm<ServiceNowFormValues>({
    resolver: zodResolver(serviceNowSchema),
    values,
  });

  if (isLoading) {
    return (
      <Stack alignItems="center" py={4}>
        <CircularProgress data-testid="servicenow-loading" />
      </Stack>
    );
  }

  if (loadError) {
    return (
      <Alert severity="error" data-testid="servicenow-load-error">
        {sepErrorMessage(loadError, Messages.serviceNow.errors.loadFailed)}
      </Alert>
    );
  }

  // A SEP build that does not carry the key at all would answer any write with
  // a 422, so there is nothing to offer — as distinct from a deployment that
  // carries it and has simply not been configured yet.
  if (!stored.isPresent) {
    return (
      <Alert severity="info" data-testid="servicenow-unavailable">
        {Messages.serviceNow.unavailable}
      </Alert>
    );
  }

  const onSubmit = async (values: ServiceNowFormValues) => {
    try {
      await patchSetting({
        settingClass: SEP_SETTINGS_CLASS,
        key: DELIVERY_INPUTS_KEY,
        value: buildDeliveryInputsPatch(values, declaredNames, stored),
      });
      enqueueSnackbar(Messages.serviceNow.saveSuccess, { variant: 'success' });
    } catch {
      // The rejected mutation is rendered inline by `saveError`; the previous
      // configuration is intact because SEP writes nothing on a failed validate.
    }
  };

  const onDisconnect = async () => {
    try {
      await resetSetting({
        settingClass: SEP_SETTINGS_CLASS,
        key: DELIVERY_INPUTS_KEY,
      });
      enqueueSnackbar(Messages.serviceNow.disconnectSuccess, {
        variant: 'success',
      });
      setDisconnectOpen(false);
    } catch (error) {
      enqueueSnackbar(sepErrorMessage(error as ApiError), {
        variant: 'error',
      });
    }
  };

  const { serviceNow } = Messages;

  return (
    <FormProvider {...methods}>
      <Stack
        component="form"
        onSubmit={methods.handleSubmit(onSubmit)}
        gap={3}
        sx={{
          [`.${formControlClasses.root}`]: {
            margin: 0,
          },
        }}
      >
        <Stack gap={1} maxWidth={MAX_LABEL_WIDTH}>
          <SettingsFieldLabel
            label={serviceNow.label}
            description={serviceNow.description}
            data-testid="servicenow-label"
          />
          <Typography variant="body2">{serviceNow.scopeNote}</Typography>
        </Stack>

        <Alert
          severity={STATUS_SEVERITY[status]}
          data-testid="servicenow-status"
          sx={{ maxWidth: MAX_LABEL_WIDTH }}
        >
          {STATUS_MESSAGE[status]}
        </Alert>

        <Stack gap={2} maxWidth={MAX_LABEL_WIDTH}>
          <TextInput
            name="endpoint"
            label={serviceNow.endpointLabel}
            textFieldProps={{
              placeholder: serviceNow.endpointPlaceholder,
              helperText: serviceNow.endpointHelper,
              slotProps: {
                htmlInput: { 'data-testid': 'servicenow-endpoint' },
              },
            }}
            formHelperTextProps={helperTextTestId('servicenow-endpoint-helper')}
          />

          <Divider />
          <Typography variant="subtitle2">
            {serviceNow.secretsLegend}
          </Typography>

          {declaredNames.length === 0 ? (
            <Typography variant="body2" data-testid="servicenow-no-secrets">
              {serviceNow.noSecrets}
            </Typography>
          ) : (
            declaredNames.map((name, index) => (
              <TextInput
                key={name}
                // Addressed by position: a declared name is runtime data from
                // SEP, and react-hook-form would read one containing a `.` as a
                // nested path.
                name={`secrets.${index}`}
                label={secretLabel(name)}
                textFieldProps={{
                  type: 'password',
                  autoComplete: 'off',
                  // A name the plan declares but the stored inputs do not carry
                  // (an image renamed it) has nothing to keep, so it gets the
                  // plain helper even though an override exists.
                  helperText: stored.secrets[name]
                    ? serviceNow.secretStoredHelper(name)
                    : serviceNow.secretHelper(name),
                  slotProps: {
                    htmlInput: { 'data-testid': `servicenow-secret-${name}` },
                  },
                }}
                formHelperTextProps={helperTextTestId(
                  `servicenow-secret-${name}-helper`
                )}
              />
            ))
          )}
        </Stack>

        {saveError && (
          <Alert
            severity="error"
            data-testid="servicenow-save-error"
            sx={{ maxWidth: MAX_LABEL_WIDTH }}
          >
            {sepErrorMessage(saveError)}
          </Alert>
        )}

        <Typography variant="body2">
          {serviceNow.subscriptionPrompt}{' '}
          <Link
            href={serviceNow.subscriptionLink}
            target="_blank"
            rel="noopener noreferrer"
          >
            {serviceNow.subscriptionLinkText}
          </Link>
        </Typography>

        <Stack
          direction="row"
          alignItems="center"
          justifyContent="space-between"
          gap={2}
          maxWidth={MAX_LABEL_WIDTH}
        >
          <SettingsSubmitButton testId="servicenow-submit" />
          {stored.hasOverride && (
            <Button
              variant="text"
              color="error"
              data-testid="servicenow-disconnect"
              onClick={() => setDisconnectOpen(true)}
            >
              {serviceNow.disconnect}
            </Button>
          )}
        </Stack>
      </Stack>

      <Modal
        open={disconnectOpen}
        onClose={() => setDisconnectOpen(false)}
        title={serviceNow.disconnectTitle}
      >
        <Stack gap={3}>
          <Typography variant="body2">{serviceNow.disconnectBody}</Typography>
          <Stack direction="row" gap={1} justifyContent="flex-end">
            <Button
              variant="text"
              onClick={() => setDisconnectOpen(false)}
              data-testid="servicenow-disconnect-cancel"
            >
              {serviceNow.disconnectCancel}
            </Button>
            <Button
              variant="contained"
              color="error"
              disabled={isDisconnecting}
              onClick={onDisconnect}
              data-testid="servicenow-disconnect-confirm"
            >
              {serviceNow.disconnectConfirm}
            </Button>
          </Stack>
        </Stack>
      </Modal>
    </FormProvider>
  );
};
