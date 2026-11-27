import Alert from '@mui/material/Alert';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useDiscoverRDS, useServerRDSInstances } from 'hooks/api/useServices';
import { enqueueSnackbar } from 'notistack';
import { type FC, useCallback, useState } from 'react';
import type {
  DiscoverRDSInstance,
  DiscoverRDSPayload,
} from 'types/services.types';
import { useBackToList } from '../../AddServicePage.context';
import { RdsServiceForm } from '../../service-form/rds';
import { DiscoveredInstances } from '../components/discovered-instances';
import { DiscoveryDocs } from '../components/discovery-docs';
import { RdsCredentials } from './rds-credentials';
import { RDS_DOC_LINKS, RDS_INSTANCE_COLUMNS } from './RdsDiscovery.constants';
import { Messages } from './RdsDiscovery.messages';
import { getRdsDiscoveryError } from './RdsDiscovery.utils';

export const RdsDiscovery: FC = () => {
  const [selected, setSelected] = useState<DiscoverRDSInstance>();
  const backToList = useCallback(() => setSelected(undefined), []);
  useBackToList(selected && backToList);
  // Lists what PMM Server's own AWS identity can see until the user discovers with credentials.
  const serverDiscovery = useServerRDSInstances();
  const { mutate, data, variables, isPending } = useDiscoverRDS();
  const instances = variables ? data : serverDiscovery.data;

  const discover = (payload: DiscoverRDSPayload) =>
    mutate(payload, {
      onError: (error) =>
        enqueueSnackbar(getRdsDiscoveryError(error), { variant: 'error' }),
    });

  if (selected) {
    return (
      <RdsServiceForm
        instance={selected}
        credentials={variables ?? {}}
        onCancel={backToList}
      />
    );
  }

  return (
    <Stack gap={4}>
      <RdsCredentials
        credentials={variables}
        onDiscover={discover}
        isDiscovering={isPending}
      />
      <DiscoveryDocs links={RDS_DOC_LINKS} />
      {!variables && serverDiscovery.isError ? (
        <Alert severity="info" data-testid="rds-server-discovery-failed">
          {Messages.serverDiscovery.failed}
        </Alert>
      ) : (
        <Stack gap={1}>
          <DiscoveredInstances
            tableName="rds-instances"
            columns={RDS_INSTANCE_COLUMNS}
            instances={instances?.rdsInstances ?? []}
            isLoading={variables ? isPending : serverDiscovery.isLoading}
            onSelect={setSelected}
          />
          {!variables && serverDiscovery.isLoading && (
            <Typography variant="body2" color="text.secondary">
              {Messages.serverDiscovery.loading}
            </Typography>
          )}
        </Stack>
      )}
    </Stack>
  );
};
