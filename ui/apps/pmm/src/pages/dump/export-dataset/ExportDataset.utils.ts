import { subHours } from 'date-fns';
import { StartDumpPayload } from 'types/dump.types';
import { ManagedService } from 'types/services.types';
import { ExportDatasetFormValues } from './ExportDataset.schema';

export interface ServiceOption {
  label: string;
  value: string;
}

export const getDefaultValues = (): ExportDatasetFormValues => {
  const endTime = new Date();
  endTime.setSeconds(0, 0);

  return {
    serviceNames: [],
    startTime: subHours(endTime, 12),
    endTime,
    exportQan: false,
    ignoreLoad: true,
    enableEncryption: false,
    encryptionPassword: '',
  };
};

export const getServiceOptions = (
  services: ManagedService[]
): ServiceOption[] =>
  Array.from(new Set(services.map(({ serviceName }) => serviceName)))
    .sort()
    .map((serviceName) => ({ label: serviceName, value: serviceName }));

export const toStartDumpPayload = (
  values: ExportDatasetFormValues
): StartDumpPayload => ({
  serviceNames: values.serviceNames,
  startTime: values.startTime.toISOString(),
  endTime: values.endTime.toISOString(),
  exportQan: values.exportQan,
  ignoreLoad: values.ignoreLoad,
  enableEncryption: values.enableEncryption,
  encryptionPassword: values.enableEncryption ? values.encryptionPassword : '',
});
