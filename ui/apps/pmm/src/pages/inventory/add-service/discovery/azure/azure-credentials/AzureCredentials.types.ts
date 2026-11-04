import type { DiscoverAzurePayload } from 'types/services.types';
import type { z } from 'zod';
import type { azureCredentialsSchema } from './AzureCredentials.schema';

export type AzureCredentialsFormValues = z.infer<typeof azureCredentialsSchema>;

export interface AzureCredentialsProps {
  onDiscover: (credentials: DiscoverAzurePayload) => void;
  isDiscovering: boolean;
}
