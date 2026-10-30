import { subHours } from 'date-fns';
import { StartDumpPayload } from 'types/dump.types';
import { ListServicesResponse } from 'types/services.types';
import { ExportDatasetFormValues } from './ExportDataset.schema';

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

export const getServiceOptions = (response: ListServicesResponse = {}) =>
  Array.from(
    new Set(
      Object.values(response).flatMap(
        (services: { serviceName: string }[] = []) =>
          services.map(({ serviceName }) => serviceName)
      )
    )
  ).sort();

export const toStartDumpPayload = (
  values: ExportDatasetFormValues
): StartDumpPayload => ({
  ...values,
  startTime: values.startTime.toISOString(),
  endTime: values.endTime.toISOString(),
  encryptionPassword: values.enableEncryption ? values.encryptionPassword : '',
});
