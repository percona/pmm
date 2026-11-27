import type {
  DiscoverAzureInstance,
  DiscoverAzurePayload,
} from 'types/services.types';
import type { z } from 'zod';
import type { azureSchema } from './AzureServiceForm.schema';

export type AzureFormValues = z.infer<typeof azureSchema>;

export interface AzureServiceFormProps {
  instance: DiscoverAzureInstance;
  credentials: DiscoverAzurePayload;
  // Returns to the discovered instances.
  onCancel: () => void;
}
