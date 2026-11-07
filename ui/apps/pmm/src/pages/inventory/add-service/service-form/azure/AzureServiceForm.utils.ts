import type {
  DiscoverAzureInstance,
  DiscoverAzurePayload,
} from 'types/services.types';
import type {
  DatabaseServiceFormValues,
  QueryAnalyticsOptionsFormValues,
} from '../ServiceForm.types';
import { toConnection, toKeyValueMap } from '../ServiceForm.utils';
import type { AzureFormValues } from './AzureServiceForm.types';

export const toAzureDefaults = (
  instance: DiscoverAzureInstance,
  defaultPort: string
) => ({
  serviceName: instance.address,
  address: instance.address,
  port: defaultPort,
  username: instance.username,
  region: instance.region,
  az: instance.az,
  azureDatabaseExporter: true,
});

// Fields every Azure database is added with, whatever its engine.
export const toAzureService = (
  values: DatabaseServiceFormValues &
    QueryAnalyticsOptionsFormValues &
    AzureFormValues,
  instance: DiscoverAzureInstance,
  credentials: DiscoverAzurePayload
) => {
  const connection = toConnection(values);

  return {
    ...connection,
    ...credentials,
    region: values.region,
    az: values.az,
    instanceId: instance.instanceId,
    nodeName: connection.serviceName,
    environment: values.environment,
    customLabels: toKeyValueMap(values.customLabels),
    azureResourceGroup: instance.azureResourceGroup,
    azureDatabaseExporter: values.azureDatabaseExporter,
    disableQueryExamples: values.disableQueryExamples,
  };
};
