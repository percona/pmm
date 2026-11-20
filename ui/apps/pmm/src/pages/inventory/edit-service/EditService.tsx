import Alert from '@mui/material/Alert';
import CircularProgress from '@mui/material/CircularProgress';
import Stack from '@mui/material/Stack';
import { Page } from 'components/page';
import { useRdsExporter } from 'hooks/api/useAgents';
import { useService } from 'hooks/api/useServices';
import type { FC } from 'react';
import { useParams } from 'react-router-dom';
import { OrgRole } from 'types/user.types';
import { EditServiceForm } from './edit-service-form';
import { Messages } from './EditService.messages';

export const EditService: FC = () => {
  const { serviceId } = useParams();
  const serviceQuery = useService(serviceId);
  const service = serviceQuery.data;
  const exporterQuery = useRdsExporter(service?.nodeId);
  // The form takes its initial values once, so it waits for fresh data rather
  // than a cached one
  const isLoaded =
    serviceQuery.isFetchedAfterMount &&
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
      {isLoaded && serviceQuery.isSuccess && !service && (
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
