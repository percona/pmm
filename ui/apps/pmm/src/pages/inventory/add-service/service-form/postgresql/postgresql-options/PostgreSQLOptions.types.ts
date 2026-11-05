import type { AddServiceType } from '../../../AddServicePage.types';

export interface PostgreSQLOptionsProps {
  // RDS and Azure take no certificates or pg_stat_monitor; Azure also no exporter options or comments parsing.
  cloud?: AddServiceType.rds | AddServiceType.azure;
}
