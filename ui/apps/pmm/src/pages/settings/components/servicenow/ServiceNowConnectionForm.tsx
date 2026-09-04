import { FC, useMemo } from 'react';
import Alert from '@mui/material/Alert';
import Button from '@mui/material/Button';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { formControlClasses } from '@mui/material';
import NorthEastIcon from '@mui/icons-material/NorthEast';
import { TextInput } from '@percona/peak-ui';
import { zodResolver } from '@hookform/resolvers/zod';
import { FormProvider, useForm } from 'react-hook-form';
import { enqueueSnackbar } from 'notistack';
import { usePatchSetting } from '@sep/api';
import { Messages } from '../../Settings.messages';
import { MAX_LABEL_WIDTH } from '../../Settings.constants';
import {
  DELIVERY_INPUTS_KEY,
  SEP_SETTINGS_CLASS,
} from './ServiceNowConnection.constants';
import { serviceNowSchema } from './ServiceNowConnectionForm.schema';
import {
  ServiceNowFormValues,
  StoredDeliveryInputs,
} from './ServiceNowConnection.types';
import {
  buildDeliveryInputsPatch,
  secretHelperText,
  secretLabel,
  sepErrorMessage,
  toFormValues,
} from './ServiceNowConnection.utils';
import { SecretField } from './SecretField';

interface Props {
  declaredNames: string[];
  stored: StoredDeliveryInputs;
  /** Whether the stored values no longer satisfy this deployment's plan. */
  isDrifted: boolean;
  /** Present only while replacing credentials that are already stored. */
  onCancel?: () => void;
  onConnected: () => void;
}

/**
 * The two steps an operator without a working connection has to walk.
 *
 * Step 1 exists because the credentials cannot be self-served: Percona Support
 * issues them per instance, and a form that only asks for them leaves anyone
 * who does not already hold them with nowhere to go.
 *
 * The write is one whole-object PATCH of `DIAGNOSTICS_DELIVERY_INPUTS` carrying
 * exactly the secret names the SEP image declares — SEP seals the leaves and
 * rejects an unexpected name, so neither is a shape the UI may improvise. All
 * validation is server-side and all-or-nothing: a rejected save leaves the
 * previous configuration standing, which is why nothing here is optimistic.
 */
export const ServiceNowConnectionForm: FC<Props> = ({
  declaredNames,
  stored,
  isDrifted,
  onCancel,
  onConnected,
}) => {
  const { mutateAsync: patchSetting, error: saveError } = usePatchSetting();
  const { serviceNow } = Messages;

  // `values` (not `defaultValues`) so a refetch — the invalidation after a save,
  // in particular — re-seeds the endpoint with what SEP actually stored. React
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
  const {
    formState: { isSubmitting, isValid },
  } = methods;

  const onSubmit = async (values: ServiceNowFormValues) => {
    try {
      await patchSetting({
        settingClass: SEP_SETTINGS_CLASS,
        key: DELIVERY_INPUTS_KEY,
        value: buildDeliveryInputsPatch(values, declaredNames),
      });
      enqueueSnackbar(serviceNow.saveSuccess, { variant: 'success' });
      onConnected();
    } catch {
      // The rejected mutation is rendered inline by `saveError`; the previous
      // configuration is intact because SEP writes nothing on a failed validate.
    }
  };

  return (
    <FormProvider {...methods}>
      <Stack
        component="form"
        onSubmit={methods.handleSubmit(onSubmit)}
        gap={4}
        maxWidth={MAX_LABEL_WIDTH}
        sx={{
          [`.${formControlClasses.root}`]: {
            margin: 0,
          },
        }}
      >
        <Stack gap={2} alignItems="flex-start">
          <Typography variant="h6" data-testid="servicenow-step-credentials">
            {serviceNow.getCredentialsStep}
          </Typography>
          <Typography variant="body2">
            {serviceNow.getCredentialsBody}
          </Typography>
          <Button
            variant="outlined"
            component="a"
            href={serviceNow.requestCredentialsLink}
            target="_blank"
            rel="noopener noreferrer"
            endIcon={<NorthEastIcon />}
            data-testid="servicenow-request-credentials"
          >
            {serviceNow.requestCredentials}
          </Button>
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
        </Stack>

        <Stack gap={2}>
          <Stack gap={1}>
            <Typography variant="h6" data-testid="servicenow-step-connect">
              {serviceNow.connectStep}
            </Typography>
            <Typography variant="body2">{serviceNow.connectBody}</Typography>
          </Stack>

          {isDrifted && (
            <Alert severity="warning" data-testid="servicenow-drifted">
              {serviceNow.driftedWarning}
            </Alert>
          )}

          {declaredNames.length === 0 ? (
            <Typography variant="body2" data-testid="servicenow-no-secrets">
              {serviceNow.noSecrets}
            </Typography>
          ) : (
            declaredNames.map((name, index) => (
              <SecretField
                key={name}
                // Addressed by position: a declared name is runtime data from
                // SEP, and react-hook-form would read one containing a `.` as a
                // nested path.
                name={`secrets.${index}`}
                label={secretLabel(name)}
                helperText={secretHelperText(name)}
                testId={`servicenow-secret-${name}`}
              />
            ))
          )}

          <TextInput
            name="endpoint"
            label={serviceNow.endpointLabel}
            textFieldProps={{
              helperText: serviceNow.endpointHelper,
              slotProps: {
                htmlInput: { 'data-testid': 'servicenow-endpoint' },
              },
            }}
          />
        </Stack>

        {saveError && (
          <Alert severity="error" data-testid="servicenow-save-error">
            {sepErrorMessage(saveError)}
          </Alert>
        )}

        <Stack direction="row" gap={1} alignItems="center">
          <Button
            type="submit"
            variant="contained"
            disabled={!isValid || isSubmitting}
            data-testid="servicenow-submit"
          >
            {isSubmitting ? serviceNow.submitting : serviceNow.submit}
          </Button>
          {onCancel && (
            <Button
              variant="text"
              onClick={onCancel}
              data-testid="servicenow-cancel"
            >
              {serviceNow.cancelRenew}
            </Button>
          )}
        </Stack>
      </Stack>
    </FormProvider>
  );
};
