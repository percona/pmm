import Stack from '@mui/material/Stack';
import { useDiscoverAzure } from 'hooks/api/useServices';
import { type FC, useState } from 'react';
import type { DiscoverAzureInstance } from 'types/services.types';
import { AzureServiceForm } from '../../service-form/azure';
import { DiscoveredInstances } from '../components/discovered-instances';
import { DiscoveryDocs } from '../components/discovery-docs';
import { AzureCredentials } from './azure-credentials';
import {
  AZURE_DOC_LINKS,
  AZURE_INSTANCE_COLUMNS,
} from './AzureDiscovery.constants';

export const AzureDiscovery: FC = () => {
  const [selected, setSelected] = useState<DiscoverAzureInstance>();
  const { mutate: discover, data, variables, isPending } = useDiscoverAzure();

  if (selected && variables) {
    return (
      <AzureServiceForm
        instance={selected}
        credentials={variables}
        onCancel={() => setSelected(undefined)}
      />
    );
  }

  return (
    <Stack gap={4}>
      <AzureCredentials
        credentials={variables}
        onDiscover={(payload) => discover(payload)}
        isDiscovering={isPending}
      />
      <DiscoveryDocs links={AZURE_DOC_LINKS} />
      <DiscoveredInstances
        tableName="azure-instances"
        columns={AZURE_INSTANCE_COLUMNS}
        instances={data?.azureDatabaseInstance ?? []}
        isLoading={isPending}
        onSelect={setSelected}
      />
    </Stack>
  );
};
