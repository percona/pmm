import Stack from '@mui/material/Stack';
import { useDiscoverRDS, useServerRDSInstances } from 'hooks/api/useServices';
import { enqueueSnackbar } from 'notistack';
import { type FC, useState } from 'react';
import type {
  DiscoverRDSInstance,
  DiscoverRDSPayload,
} from 'types/services.types';
import { RdsServiceForm } from '../../service-form/rds';
import { DiscoveredInstances } from '../components/discovered-instances';
import { DiscoveryDocs } from '../components/discovery-docs';
import { RdsCredentials } from './rds-credentials';
import { RDS_DOC_LINKS, RDS_INSTANCE_COLUMNS } from './RdsDiscovery.constants';
import { getRdsDiscoveryError } from './RdsDiscovery.utils';

export const RdsDiscovery: FC = () => {
  const [selected, setSelected] = useState<DiscoverRDSInstance>();
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
        onCancel={() => setSelected(undefined)}
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
      <DiscoveredInstances
        tableName="rds-instances"
        columns={RDS_INSTANCE_COLUMNS}
        instances={instances?.rdsInstances ?? []}
        isLoading={variables ? isPending : serverDiscovery.isLoading}
        onSelect={setSelected}
      />
    </Stack>
  );
};
