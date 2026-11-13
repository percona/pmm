import type { DiscoverAzurePayload } from 'types/services.types';
import type { z } from 'zod';
import type { azureCredentialsSchema } from './AzureCredentials.schema';

export type AzureCredentialsFormValues = z.infer<typeof azureCredentialsSchema>;

export interface AzureCredentialsProps {
  // The last discovery's credentials, shown again when coming back from an instance.
  credentials?: DiscoverAzurePayload;
  onDiscover: (credentials: DiscoverAzurePayload) => void;
  isDiscovering: boolean;
}
