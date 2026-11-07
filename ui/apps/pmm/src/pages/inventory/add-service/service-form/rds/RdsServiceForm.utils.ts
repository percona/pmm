import type {
  DiscoverRDSInstance,
  DiscoverRDSPayload,
} from 'types/services.types';
import type {
  DatabaseServiceFormValues,
  QueryAnalyticsOptionsFormValues,
} from '../ServiceForm.types';
import { getMetricsMode, toConnection, toLabels } from '../ServiceForm.utils';
import type { RdsFormValues } from './RdsServiceForm.types';

export const toRdsDefaults = (
  instance: DiscoverRDSInstance,
  defaultPort: string
) => ({
  serviceName: instance.instanceId,
  address: instance.address,
  port: instance.port ? String(instance.port) : defaultPort,
  region: instance.region,
  az: instance.az,
  instanceId: instance.instanceId,
  disableBasicMetrics: false,
  disableEnhancedMetrics: false,
});

// Fields every RDS service is added with, whatever its engine.
export const toRdsService = (
  values: DatabaseServiceFormValues &
    QueryAnalyticsOptionsFormValues &
    RdsFormValues,
  instance: DiscoverRDSInstance,
  credentials: DiscoverRDSPayload
) => {
  const connection = toConnection(values);

  return {
    ...connection,
    ...credentials,
    ...toLabels(values),
    region: values.region,
    az: values.az,
    instanceId: values.instanceId,
    engine: instance.engine,
    nodeName: connection.serviceName,
    rdsExporter: true,
    disableQueryExamples: values.disableQueryExamples,
    disableCommentsParsing: values.disableCommentsParsing,
    disableBasicMetrics: values.disableBasicMetrics,
    disableEnhancedMetrics: values.disableEnhancedMetrics,
    metricsMode: getMetricsMode(values.isPmmServerNode),
  };
};
