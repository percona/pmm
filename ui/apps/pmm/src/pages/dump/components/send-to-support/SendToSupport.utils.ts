import { UploadDumpsPayload } from 'types/dump.types';
import { SendToSupportFormValues } from './SendToSupport.schema';

export const SEND_TO_SUPPORT_DEFAULT_VALUES: SendToSupportFormValues = {
  address: 'sftp.percona.com:2222',
  user: '',
  password: '',
  directory: '',
};

export const toUploadPayload = (
  values: SendToSupportFormValues,
  dumpIds: string[]
): UploadDumpsPayload => ({
  dumpIds,
  sftpParameters: values,
});
