import type { TrackingOption } from '../PostgreSQLForm.types';

export interface PostgreSQLOptionsProps {
  trackingOptions?: { value: TrackingOption; label: string }[];
  // The exporter's collectors and connection limit; Azure has neither.
  withExporterOptions?: boolean;
}
