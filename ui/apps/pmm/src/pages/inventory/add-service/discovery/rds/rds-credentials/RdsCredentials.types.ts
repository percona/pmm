import type { DiscoverRDSPayload } from 'types/services.types';
import type { z } from 'zod';
import type { rdsCredentialsSchema } from './RdsCredentials.schema';

export type RdsCredentialsFormValues = z.infer<typeof rdsCredentialsSchema>;

export interface RdsCredentialsProps {
  // The last discovery's credentials, shown again when coming back from an instance.
  credentials?: DiscoverRDSPayload;
  onDiscover: (credentials: DiscoverRDSPayload) => void;
  isDiscovering: boolean;
}
