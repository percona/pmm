import { subHours } from 'date-fns';
import type { StartDumpPayload } from 'types/dump.types';
import type { ListServicesResponse } from 'types/services.types';
import type { ExportDatasetFormValues } from './ExportDataset.schema';
import { fromWallClock, toWallClock } from './ExportDataset.time';

export { fromWallClock, toWallClock };

export const getDefaultValues = (timezone: string): ExportDatasetFormValues => {
  const endTime = toWallClock(new Date(), timezone);
  endTime.setSeconds(0, 0);

  return {
    serviceScope: 'all',
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
  { serviceScope, serviceNames, ...values }: ExportDatasetFormValues,
  timezone: string
): StartDumpPayload => ({
  ...values,
  serviceNames: serviceScope === 'all' ? [] : serviceNames,
  startTime: fromWallClock(values.startTime, timezone).toISOString(),
  endTime: fromWallClock(values.endTime, timezone).toISOString(),
  encryptionPassword: values.enableEncryption ? values.encryptionPassword : '',
});
