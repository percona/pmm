import { subHours } from 'date-fns';
import type { StartDumpPayload } from 'types/dump.types';
import type { ListServicesResponse } from 'types/services.types';
import type { ExportDatasetFormValues } from './ExportDataset.schema';

export const getDefaultValues = (): ExportDatasetFormValues => {
  const endTime = new Date();
  endTime.setSeconds(0, 0);

  return {
    serviceScope: 'all',
    serviceNames: [],
    startTime: subHours(endTime, 12),
    endTime,
    exportQan: false,
    ignoreLoad: false,
    enableEncryption: false,
    encryptionPassword: '',
  };
};

export const getServiceOptions = (response: ListServicesResponse = {}) =>
  Array.from(
    new Set(
      Object.values(response).flatMap(
        (services: { serviceName: string }[] = []) =>
          services.map(({ serviceName }) => serviceName)
      )
    )
  ).sort();

export const toStartDumpPayload = ({
  serviceScope,
  serviceNames,
  ...values
}: ExportDatasetFormValues): StartDumpPayload => ({
  ...values,
  serviceNames: serviceScope === 'all' ? [] : serviceNames,
  startTime: values.startTime.toISOString(),
  endTime: values.endTime.toISOString(),
  encryptionPassword: values.enableEncryption ? values.encryptionPassword : '',
});
