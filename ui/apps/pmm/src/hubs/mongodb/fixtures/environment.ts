import { matchesEnvironment } from 'hubs/template/hub.params';
import type { MongoSnapshot } from '../mongodb.types';

// Where each sample query and install ran; live QAN and Operations filter by these labels on their side.
const QUERY_ENVIRONMENTS: Record<string, string> = {
  'q-orders-find': 'prod',
  'q-billing-update': 'prod',
};

const INSTALL_ENVIRONMENTS: Record<string, string> = {
  'run-install-pmm-client': 'prod',
};

/** A sample snapshot as the live sources return it for one environment: QAN rows, Operations counts and installs narrowed, the rest left to the derivation. */
export const forEnvironment = (
  snapshot: MongoSnapshot,
  environment: string | null
): MongoSnapshot => {
  const shown = (env: string | undefined) =>
    matchesEnvironment(env, environment);
  const services = (snapshot.services ?? []).filter((service) =>
    shown(service.environment)
  );
  const { qan, operations } = snapshot;

  return {
    ...snapshot,
    qan: qan && {
      current: qan.current.filter((row) =>
        shown(QUERY_ENVIRONMENTS[row.queryId])
      ),
      baseline: qan.baseline.filter((row) =>
        shown(QUERY_ENVIRONMENTS[row.queryId])
      ),
    },
    operations: operations && {
      ...operations,
      // Operations counts a service without an up series as down.
      summary: {
        ...operations.summary,
        services: services.length,
        up: services.filter(
          (service) => snapshot.metrics?.up[service.serviceId]?.value === 1
        ).length,
      },
      // A scan covers every host, whatever the environment.
      failedRuns: operations.failedRuns.filter(
        (run) => run.kind === 'scan' || shown(INSTALL_ENVIRONMENTS[run.id])
      ),
    },
  };
};
