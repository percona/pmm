import type {
  DiscoverRDSInstance,
  DiscoverRDSPayload,
} from 'types/services.types';
import type { z } from 'zod';
import type { rdsSchema } from './RdsServiceForm.schema';

export type RdsFormValues = z.infer<typeof rdsSchema>;

export interface RdsServiceFormProps {
  instance: DiscoverRDSInstance;
  credentials: DiscoverRDSPayload;
  // Returns to the discovered instances.
  onCancel: () => void;
}
