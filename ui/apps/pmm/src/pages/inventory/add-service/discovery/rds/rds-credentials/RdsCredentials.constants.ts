import { RdsAuthMode } from 'pages/inventory/Inventory.types';
import { Messages } from './RdsCredentials.messages';

const { authMode } = Messages;

export const RDS_AUTH_MODE_OPTIONS = [
  { value: RdsAuthMode.accessKey, label: authMode.accessKey },
  { value: RdsAuthMode.iamRole, label: authMode.iamRole },
  { value: RdsAuthMode.hostCredentials, label: authMode.serverIdentity },
];
