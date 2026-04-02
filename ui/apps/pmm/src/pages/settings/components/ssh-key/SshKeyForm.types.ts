import { Settings } from 'types/settings.types';

export interface SshKeyFormProps {
  settings: Settings;
}

export interface SshKeyFormValues {
  sshKey: string;
}
