import type { AddServiceType } from '../../../AddServicePage.types';

export interface MySQLOptionsProps {
  // RDS and Azure take no certificates or extra DSN params; Azure also no collectors or comments parsing.
  cloud?: AddServiceType.rds | AddServiceType.azure;
}
