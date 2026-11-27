import type { z } from 'zod';
import type {
  postgresqlOptionsSchema,
  postgresqlSchema,
} from './PostgreSQLForm.schema';

export enum TrackingOption {
  none = 'none',
  pgStatements = 'pgStatements',
  pgMonitor = 'pgMonitor',
}

export enum AutoDiscoveryOption {
  enabled = 'enabled',
  disabled = 'disabled',
  custom = 'custom',
}

export enum MaxConnectionsOption {
  enabled = 'enabled',
  disabled = 'disabled',
}

export type PostgreSQLOptionsFormValues = z.infer<
  typeof postgresqlOptionsSchema
>;
export type PostgreSQLFormValues = z.infer<typeof postgresqlSchema>;
