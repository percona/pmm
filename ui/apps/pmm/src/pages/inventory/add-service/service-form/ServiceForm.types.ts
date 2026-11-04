import type { ReactNode } from 'react';
import type { FieldValues, UseFormReturn } from 'react-hook-form';
import type { z } from 'zod';
import type {
  commonOptionsSchema,
  connectionSchema,
  databaseServiceSchema,
  disableCollectorsSchema,
  labelsSchema,
  maxQueryLengthSchema,
  mongodbTlsCertificatesSchema,
  nodeSchema,
  queryAnalyticsOptionsSchema,
  tlsCertificatesSchema,
  tlsSchema,
} from './ServiceForm.schema';

export type ConnectionFormValues = z.infer<typeof connectionSchema>;
export type NodeFormValues = z.infer<typeof nodeSchema>;
export type LabelsFormValues = z.infer<typeof labelsSchema>;
export type CommonOptionsFormValues = z.infer<typeof commonOptionsSchema>;
export type TlsFormValues = z.infer<typeof tlsSchema>;
export type TlsCertificatesFormValues = z.infer<typeof tlsCertificatesSchema>;
export type MongodbTlsCertificatesFormValues = z.infer<
  typeof mongodbTlsCertificatesSchema
>;
export type QueryAnalyticsOptionsFormValues = z.infer<
  typeof queryAnalyticsOptionsSchema
>;
export type DisableCollectorsFormValues = z.infer<
  typeof disableCollectorsSchema
>;

export type MaxQueryLengthFormValues = z.infer<typeof maxQueryLengthSchema>;

export type DatabaseServiceFormValues = z.infer<typeof databaseServiceSchema>;

export interface ServiceFormProps<
  T extends FieldValues & Pick<ConnectionFormValues, 'serviceName' | 'address'>,
> {
  form: UseFormReturn<T>;
  // Shown in the success message, e.g. "MySQL".
  serviceTypeLabel: string;
  onSubmit: (values: T) => Promise<void>;
  // Cancel goes back to the service type selection unless this handles it.
  onCancel?: () => void;
  children: ReactNode;
}
