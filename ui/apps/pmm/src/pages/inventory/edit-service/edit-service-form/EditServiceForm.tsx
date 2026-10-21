import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import { TextInput } from '@percona/peak-ui';
import { zodResolver } from '@hookform/resolvers/zod';
import { useChangeRdsCredentials } from 'hooks/api/useAgents';
import { useUpdateService } from 'hooks/api/useServices';
import { PMM_INVENTORY_SERVICES_PATH } from 'lib/constants';
import { enqueueSnackbar } from 'notistack';
import { type FC, useMemo, useState } from 'react';
import { FormProvider, useForm } from 'react-hook-form';
import { Link as RouterLink, useNavigate } from 'react-router-dom';
import type { RdsExporter } from 'types/agent.types';
import type { EditableService } from 'types/services.types';
import { ConfirmChangesModal } from '../confirm-changes-modal';
import { ConnectionSettings } from '../connection-settings';
import { SERVICE_SETTINGS } from '../EditService.constants';
import { Messages } from '../EditService.messages';
import {
  type EditServiceFormValues,
  getEditServiceSchema,
} from '../EditService.schema';
import { RdsAuthMode } from '../EditService.types';
import {
  getExporter,
  getInitialValues,
  LABEL_CHANGES,
  toRdsCredentials,
  toServiceUpdate,
} from '../EditService.utils';
import { MetricsSettings } from '../metrics-settings';
import { QueryAnalyticsSettings } from '../query-analytics-settings';
import { RdsCredentials } from '../rds-credentials';
import { SectionHeader } from '../section-header';
import { ServiceDetails } from '../service-details';
import { TlsSettings } from '../tls-settings';

const { labels } = Messages;

interface Props {
  editable: EditableService;
  rdsExporter: RdsExporter | null;
  nodeName: string;
  isCloudNode: boolean;
}

export const EditServiceForm: FC<Props> = ({
  editable,
  rdsExporter,
  nodeName,
  isCloudNode,
}) => {
  const { service } = editable;
  const { serviceType } = service;
  const settings = SERVICE_SETTINGS[serviceType];
  const navigate = useNavigate();
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);
  const { mutateAsync: updateService } = useUpdateService();
  const { mutateAsync: changeRdsCredentials } = useChangeRdsCredentials();
  // Changes are what differs from these, so a refetch must not move them
  const [initialValues] = useState(() =>
    getInitialValues(editable, rdsExporter)
  );
  const [exporter] = useState(() => getExporter(editable));
  const resolver = useMemo(
    () =>
      zodResolver(
        getEditServiceSchema(serviceType, initialValues, rdsExporter)
      ),
    [serviceType, initialValues, rdsExporter]
  );
  const methods = useForm<EditServiceFormValues>({
    resolver,
    defaultValues: initialValues,
  });
  const values = methods.watch();
  const credentials = toRdsCredentials(values, rdsExporter);
  const update = toServiceUpdate(service, initialValues, values);
  const changedNames = Object.keys(update?.changes ?? {});

  // Credentials go first: the server can reject them on their own, and nothing
  // is saved then. The API error notifications report what failed.
  const handleSave = async (skipConnectionCheck: boolean) => {
    if (credentials && rdsExporter) {
      try {
        await changeRdsCredentials({
          agentId: rdsExporter.agentId,
          credentials,
        });
      } catch {
        setIsConfirmOpen(false);
        return;
      }
    }

    try {
      if (update) {
        const warning = await updateService({
          ...update,
          changes: {
            ...update.changes,
            ...(skipConnectionCheck && { skipConnectionCheck }),
          },
        });

        if (warning) {
          enqueueSnackbar(warning, { variant: 'warning' });
        }
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
        <ServiceDetails service={service} nodeName={nodeName} />
        <Stack gap={2}>
          <SectionHeader
            title={labels.title}
            description={labels.description}
          />
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
            {settings.includes('group') && (
              <TextInput<EditServiceFormValues>
                name="group"
                label={labels.group}
                textFieldProps={{ placeholder: labels.group, fullWidth: true }}
              />
            )}
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
        <ConnectionSettings serviceType={serviceType} exporter={exporter} />
        {settings.includes('tls') && (
          <TlsSettings serviceType={serviceType} exporter={exporter} />
        )}
        {settings.includes('querySource') && (
          <QueryAnalyticsSettings
            serviceType={serviceType}
            serviceName={service.serviceName}
            isCloudNode={isCloudNode}
          />
        )}
        <MetricsSettings serviceType={serviceType} />
        {rdsExporter && <RdsCredentials exporter={rdsExporter} />}
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
        labelsChanged={changedNames.some((name) => LABEL_CHANGES.has(name))}
        settingsChanged={changedNames.some((name) => !LABEL_CHANGES.has(name))}
        clusterChanged={changedNames.includes('cluster')}
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
