import type { MRT_ColumnDef } from 'material-react-table';
import {
  DiscoverRDSEngine,
  type DiscoverRDSInstance,
} from 'types/services.types';
import { Messages as PageMessages } from '../../AddServicePage.messages';
import { Messages as InstancesMessages } from '../components/discovered-instances/DiscoveredInstances.messages';
import { Messages } from './RdsDiscovery.messages';

export const RDS_DOC_LINKS = [
  {
    label: Messages.docs.credentials,
    url: 'https://per.co.na/iam_user_with_permission',
  },
  { label: Messages.docs.iamRole, url: 'https://per.co.na/iam_role' },
];

// How the AWS SDK reports that neither keys nor an IAM role give PMM Server credentials.
export const AWS_NO_CREDENTIALS_ERROR = 'failed to retrieve credentials';

const RDS_ENGINE_LABELS: Record<DiscoverRDSEngine, string> = {
  [DiscoverRDSEngine.mysql]: PageMessages.serviceTypes.mysql,
  [DiscoverRDSEngine.postgresql]: PageMessages.serviceTypes.postgresql,
  [DiscoverRDSEngine.unspecified]: InstancesMessages.unknownEngine,
};

export const RDS_INSTANCE_COLUMNS: MRT_ColumnDef<DiscoverRDSInstance>[] = [
  { accessorKey: 'region', header: Messages.columns.region },
  { accessorKey: 'az', header: Messages.columns.az },
  {
    id: 'engine',
    header: Messages.columns.engine,
    accessorFn: ({ engine, engineVersion }) =>
      `${RDS_ENGINE_LABELS[engine]} ${engineVersion}`,
  },
  { accessorKey: 'instanceId', header: Messages.columns.instanceId },
  {
    id: 'address',
    header: Messages.columns.address,
    // The address can carry the port, which has its own field.
    accessorFn: ({ address }) => address.split(':')[0],
  },
];
