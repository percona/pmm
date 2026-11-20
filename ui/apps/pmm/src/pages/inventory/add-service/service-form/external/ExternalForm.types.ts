import type { z } from 'zod';
import type { externalSchema } from './ExternalForm.schema';

export enum MetricsParameters {
  manually = 'manually',
  parsed = 'parsed',
}

export enum Scheme {
  https = 'https',
  http = 'http',
}

export type ExternalFormValues = z.infer<typeof externalSchema>;

export interface ExporterEndpoint {
  scheme: Scheme;
  address: string;
  listenPort: number;
  metricsPath?: string;
  username: string;
  password: string;
}
