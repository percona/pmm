import { DOCS_URLS } from 'lib/constants';
import type { MRT_ColumnDef } from 'material-react-table';
import {
  DiscoverAzureDatabaseType,
  type DiscoverAzureInstance,
} from 'types/services.types';
import { Messages as PageMessages } from '../../AddServicePage.messages';
import { Messages as InstancesMessages } from '../components/discovered-instances/DiscoveredInstances.messages';
import { HostName } from '../components/host-name';
import { Messages } from './AzureDiscovery.messages';

export const AZURE_DOC_LINKS = [
  { label: Messages.docs.credentials, url: DOCS_URLS.azureCredentials },
];

const AZURE_ENGINE_LABELS: Record<DiscoverAzureDatabaseType, string> = {
  [DiscoverAzureDatabaseType.mysql]: PageMessages.serviceTypes.mysql,
  [DiscoverAzureDatabaseType.postgresql]: PageMessages.serviceTypes.postgresql,
  [DiscoverAzureDatabaseType.unspecified]: InstancesMessages.unknownEngine,
};

// Sizes are minimums that, with the action column, fit the page content from a
// 1100px window up; the spare width goes to the columns with long values.
export const AZURE_INSTANCE_COLUMNS: MRT_ColumnDef<DiscoverAzureInstance>[] = [
  {
    accessorKey: 'region',
    header: Messages.columns.region,
    size: 110,
    grow: false,
  },
  {
    accessorKey: 'azureResourceGroup',
    header: Messages.columns.resourceGroup,
    size: 130,
  },
  { accessorKey: 'serviceName', header: Messages.columns.name, size: 90 },
  {
    id: 'engine',
    header: Messages.columns.engine,
    size: 110,
    grow: false,
    accessorFn: ({ type }) => AZURE_ENGINE_LABELS[type],
  },
  {
    accessorKey: 'address',
    header: Messages.columns.address,
    size: 130,
    Cell: ({ cell }) => <HostName host={cell.getValue<string>()} />,
  },
];
