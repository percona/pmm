import { DOCS_URLS } from 'lib/constants';
import type { MRT_ColumnDef } from 'material-react-table';
import {
  DiscoverRDSEngine,
  type DiscoverRDSInstance,
} from 'types/services.types';
import { Messages as PageMessages } from '../../AddServicePage.messages';
import { Messages as InstancesMessages } from '../components/discovered-instances/DiscoveredInstances.messages';
import { HostName } from '../components/host-name';
import { Messages } from './RdsDiscovery.messages';

export const RDS_DOC_LINKS = [
  { label: Messages.docs.credentials, url: DOCS_URLS.rdsCredentials },
  { label: Messages.docs.iamRole, url: DOCS_URLS.rdsIamRole },
];

// How the AWS SDK reports that neither keys nor an IAM role give PMM Server credentials.
export const AWS_NO_CREDENTIALS_ERROR = 'failed to retrieve credentials';

const RDS_ENGINE_LABELS: Record<DiscoverRDSEngine, string> = {
  [DiscoverRDSEngine.mysql]: PageMessages.serviceTypes.mysql,
  [DiscoverRDSEngine.postgresql]: PageMessages.serviceTypes.postgresql,
  [DiscoverRDSEngine.unspecified]: InstancesMessages.unknownEngine,
};

// Sizes are minimums that, with the action column, fit the page content from a
// 1100px window up; the spare width goes to the columns with long values.
export const RDS_INSTANCE_COLUMNS: MRT_ColumnDef<DiscoverRDSInstance>[] = [
  {
    accessorKey: 'region',
    header: Messages.columns.region,
    size: 110,
    grow: false,
  },
  { accessorKey: 'az', header: Messages.columns.az, size: 150, grow: false },
  {
    id: 'engine',
    header: Messages.columns.engine,
    size: 110,
    grow: false,
    accessorFn: ({ engine, engineVersion }) =>
      `${RDS_ENGINE_LABELS[engine]} ${engineVersion}`,
  },
  {
    accessorKey: 'instanceId',
    header: Messages.columns.instanceId,
    size: 90,
  },
  {
    id: 'address',
    header: Messages.columns.address,
    size: 110,
    // The address can carry the port, which has its own field.
    accessorFn: ({ address }) => address.split(':')[0],
    Cell: ({ cell }) => <HostName host={cell.getValue<string>()} />,
  },
];
