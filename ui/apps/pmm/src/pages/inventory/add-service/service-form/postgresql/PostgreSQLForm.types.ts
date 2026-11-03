import type { z } from 'zod';
import type { postgresqlSchema } from './PostgreSQLForm.schema';

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

export type PostgreSQLFormValues = z.infer<typeof postgresqlSchema>;
