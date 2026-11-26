import { Messages } from '../EditService.messages';
import { RdsAuthMode } from '../EditService.types';

const { rds } = Messages;

export const AUTH_MODE_OPTIONS = [
  { value: RdsAuthMode.accessKey, label: rds.authMode.accessKey },
  { value: RdsAuthMode.iamRole, label: rds.authMode.iamRole },
  { value: RdsAuthMode.hostCredentials, label: rds.authMode.hostCredentials },
];
