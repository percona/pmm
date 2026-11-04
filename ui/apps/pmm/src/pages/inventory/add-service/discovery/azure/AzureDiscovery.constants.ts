import type { MRT_ColumnDef } from 'material-react-table';
import {
  DiscoverAzureDatabaseType,
  type DiscoverAzureInstance,
} from 'types/services.types';
import { Messages as PageMessages } from '../../AddServicePage.messages';
import { Messages as InstancesMessages } from '../components/discovered-instances/DiscoveredInstances.messages';
import { Messages } from './AzureDiscovery.messages';

export const AZURE_DOC_LINKS = [
  { label: Messages.docs.credentials, url: 'https://per.co.na/QcYQRx' },
];

const AZURE_ENGINE_LABELS: Record<DiscoverAzureDatabaseType, string> = {
  [DiscoverAzureDatabaseType.mysql]: PageMessages.serviceTypes.mysql,
  [DiscoverAzureDatabaseType.postgresql]: PageMessages.serviceTypes.postgresql,
  [DiscoverAzureDatabaseType.unspecified]: InstancesMessages.unknownEngine,
};

export const AZURE_INSTANCE_COLUMNS: MRT_ColumnDef<DiscoverAzureInstance>[] = [
  { accessorKey: 'region', header: Messages.columns.region },
  { accessorKey: 'azureResourceGroup', header: Messages.columns.resourceGroup },
  { accessorKey: 'serviceName', header: Messages.columns.name },
  {
    id: 'engine',
    header: Messages.columns.engine,
    accessorFn: ({ type }) => AZURE_ENGINE_LABELS[type],
  },
  { accessorKey: 'address', header: Messages.columns.address },
];
