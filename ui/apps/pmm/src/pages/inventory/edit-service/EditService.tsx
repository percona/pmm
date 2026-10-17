import Alert from '@mui/material/Alert';
import CircularProgress from '@mui/material/CircularProgress';
import Stack from '@mui/material/Stack';
import { Page } from 'components/page';
import { useRdsExporter } from 'hooks/api/useAgents';
import { useManagedServices } from 'hooks/api/useServices';
import type { FC } from 'react';
import { useParams } from 'react-router-dom';
import { OrgRole } from 'types/user.types';
import { EditServiceForm } from './edit-service-form';
import { Messages } from './EditService.messages';

export const EditService: FC = () => {
  const { serviceId } = useParams();
  const servicesQuery = useManagedServices();
  const service = servicesQuery.data?.services.find(
    (service) => service.serviceId === serviceId
  );
  const exporterQuery = useRdsExporter(service?.nodeId);
  // The form takes its initial values once, so it waits for fresh data rather
  // than a cached list
  const isLoaded =
    servicesQuery.isFetchedAfterMount &&
    (!service || exporterQuery.isFetchedAfterMount);

  return (
    <Page
      title={
        service ? Messages.title(service.serviceName) : Messages.loadingTitle
      }
      roles={[OrgRole.Admin]}
    >
      {!isLoaded && (
        <Stack alignItems="center" py={4}>
          <CircularProgress data-testid="edit-service-loading" />
        </Stack>
      )}
      {isLoaded && servicesQuery.isSuccess && !service && (
        <Alert severity="error">{Messages.notFound}</Alert>
      )}
      {isLoaded && service && (
        <EditServiceForm
          service={service}
          exporter={exporterQuery.data ?? null}
        />
      )}
    </Page>
  );
};
