import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { TextInput } from '@percona/peak-ui';
import { zodResolver } from '@hookform/resolvers/zod';
import { useChangeRdsCredentials } from 'hooks/api/useAgents';
import { useChangeService } from 'hooks/api/useServices';
import { PMM_INVENTORY_SERVICES_PATH } from 'lib/constants';
import { enqueueSnackbar } from 'notistack';
import { type FC, useMemo, useState } from 'react';
import { FormProvider, useForm } from 'react-hook-form';
import { Link as RouterLink, useNavigate } from 'react-router-dom';
import type { RdsExporter } from 'types/agent.types';
import type { ManagedService } from 'types/services.types';
import { ConfirmChangesModal } from '../confirm-changes-modal';
import { Messages } from '../EditService.messages';
import {
  type EditServiceFormValues,
  getEditServiceSchema,
} from '../EditService.schema';
import { RdsAuthMode } from '../EditService.types';
import {
  getInitialValues,
  toCustomLabelsText,
  toRdsCredentials,
  toServiceChanges,
} from '../EditService.utils';
import { RdsCredentials } from '../rds-credentials';

const { labels } = Messages;

interface Props {
  service: ManagedService;
  exporter: RdsExporter | null;
}

export const EditServiceForm: FC<Props> = ({ service, exporter }) => {
  const navigate = useNavigate();
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);
  const { mutateAsync: changeService } = useChangeService();
  const { mutateAsync: changeRdsCredentials } = useChangeRdsCredentials();
  const resolver = useMemo(
    () =>
      zodResolver(
        getEditServiceSchema(exporter, toCustomLabelsText(service.customLabels))
      ),
    [exporter, service.customLabels]
  );
  const methods = useForm<EditServiceFormValues>({
    resolver,
    defaultValues: getInitialValues(service, exporter),
  });
  const values = methods.watch();
  const credentials = toRdsCredentials(values, exporter);

  // Credentials go first: the server can reject them on their own, and nothing
  // is saved then. The API error notifications report what failed.
  const handleSave = async () => {
    if (credentials && exporter) {
      try {
        await changeRdsCredentials({ agentId: exporter.agentId, credentials });
      } catch {
        setIsConfirmOpen(false);
        return;
      }
    }

    try {
      const changes = toServiceChanges(service, values);

      if (changes) {
        await changeService(changes);
      }

      enqueueSnackbar(Messages.success(service.serviceName), {
        variant: 'success',
      });
      navigate(PMM_INVENTORY_SERVICES_PATH);
    } catch {
      if (credentials) {
        enqueueSnackbar(Messages.partial, { variant: 'warning' });
      }

      setIsConfirmOpen(false);
    }
  };

  return (
    <FormProvider {...methods}>
      <Stack
        component="form"
        onSubmit={methods.handleSubmit(() => setIsConfirmOpen(true))}
        gap={4}
        sx={{ flex: 1 }}
        data-testid="edit-service-form"
      >
        <Stack gap={2}>
          <Stack gap={0.5}>
            <Typography variant="h6">{labels.title}</Typography>
            <Typography variant="body2">{labels.description}</Typography>
          </Stack>
          <Stack direction={{ xs: 'column', md: 'row' }} gap={2}>
            <TextInput<EditServiceFormValues>
              name="environment"
              label={labels.environment}
              textFieldProps={{
                placeholder: labels.environment,
                fullWidth: true,
              }}
            />
            <TextInput<EditServiceFormValues>
              name="cluster"
              label={labels.cluster}
              textFieldProps={{ placeholder: labels.cluster, fullWidth: true }}
            />
            <TextInput<EditServiceFormValues>
              name="replicationSet"
              label={labels.replicationSet}
              textFieldProps={{
                placeholder: labels.replicationSet,
                fullWidth: true,
              }}
            />
          </Stack>
          <TextInput<EditServiceFormValues>
            name="customLabels"
            label={labels.customLabels}
            textFieldProps={{
              multiline: true,
              minRows: 4,
              fullWidth: true,
              placeholder: labels.customLabelsPlaceholder,
              helperText: labels.customLabelsDescription,
            }}
          />
        </Stack>
        {exporter && <RdsCredentials exporter={exporter} />}
        <Stack
          direction="row"
          gap={2}
          sx={{
            position: 'sticky',
            bottom: 0,
            py: 2,
            mt: 'auto',
            zIndex: 1,
            bgcolor: 'background.paper',
            borderTop: 1,
            borderColor: 'divider',
          }}
        >
          <Button
            variant="contained"
            type="submit"
            disabled={!methods.formState.isDirty}
            data-testid="edit-service-submit"
          >
            {Messages.saveChanges}
          </Button>
          <Button
            component={RouterLink}
            to={PMM_INVENTORY_SERVICES_PATH}
            data-testid="edit-service-cancel"
          >
            {Messages.cancel}
          </Button>
        </Stack>
      </Stack>
      <ConfirmChangesModal
        open={isConfirmOpen}
        serviceName={service.serviceName}
        clusterChanged={values.cluster !== service.cluster}
        credentialsChanged={!!credentials}
        dropsCredentials={
          !!credentials && values.rdsAuthMode === RdsAuthMode.hostCredentials
        }
        onConfirm={handleSave}
        onClose={() => setIsConfirmOpen(false)}
      />
    </FormProvider>
  );
};
