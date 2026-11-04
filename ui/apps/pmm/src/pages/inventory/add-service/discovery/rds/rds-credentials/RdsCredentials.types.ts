import type { DiscoverRDSPayload } from 'types/services.types';
import type { z } from 'zod';
import type { rdsCredentialsSchema } from './RdsCredentials.schema';

export type RdsCredentialsFormValues = z.infer<typeof rdsCredentialsSchema>;

export interface RdsCredentialsProps {
  onDiscover: (credentials: DiscoverRDSPayload) => void;
  isDiscovering: boolean;
}
