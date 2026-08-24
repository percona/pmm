import type { DatabasesType } from '../Details.types';
import type { FetchExplainsResult } from '../Explain/Explain.types';

export interface TableContainerProps extends FetchExplainsResult {
  databaseType: DatabasesType;
  database?: string;
  example?: any;
}
