import type { z } from 'zod';
import type { mongodbSchema } from './MongoDBForm.schema';

export type MongoDBFormValues = z.infer<typeof mongodbSchema>;
