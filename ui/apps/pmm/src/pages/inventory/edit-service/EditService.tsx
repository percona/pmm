import Alert from '@mui/material/Alert';
import CircularProgress from '@mui/material/CircularProgress';
import Stack from '@mui/material/Stack';
import { Page } from 'components/page';
import { useRdsExporter } from 'hooks/api/useAgents';
import { useNode } from 'hooks/api/useNodes';
import { useEditableService } from 'hooks/api/useServices';
import type { FC } from 'react';
import { useParams } from 'react-router-dom';
import { NodeType } from 'types/nodes.types';
import { OrgRole } from 'types/user.types';
import { EditServiceForm } from './edit-service-form';
import { Messages } from './EditService.messages';

const CLOUD_NODE_TYPES = [NodeType.remoteRds, NodeType.remoteAzureDatabase];

export const EditService: FC = () => {
  const { serviceId } = useParams();
  const serviceQuery = useEditableService(serviceId);
  const editable = serviceQuery.data;
  const service = editable?.service;
  const exporterQuery = useRdsExporter(service?.nodeId);
  const nodeQuery = useNode(service?.nodeId);
  const node = nodeQuery.data;
  // The form takes its initial values once, so it waits for fresh data rather
  // than a cached one
  const isLoaded =
    serviceQuery.isFetchedAfterMount &&
    (!service || (exporterQuery.isFetchedAfterMount && nodeQuery.isFetched));

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
      {isLoaded && serviceQuery.isSuccess && !editable && (
        <Alert severity="error">{Messages.notFound}</Alert>
      )}
      {isLoaded && editable && (
        <EditServiceForm
          editable={editable}
          rdsExporter={exporterQuery.data ?? null}
          nodeName={node?.nodeName ?? editable.service.nodeId}
          isCloudNode={!!node && CLOUD_NODE_TYPES.includes(node.nodeType)}
        />
      )}
    </Page>
  );
};
