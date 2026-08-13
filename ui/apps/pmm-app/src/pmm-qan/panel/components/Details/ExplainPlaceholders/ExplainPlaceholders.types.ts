import type { ReactNode } from 'react';
import type { DatabasesType, QueryExampleResponseItem } from '../Details.types';
import type { FetchExplainsResult } from '../Explain/Explain.types';

export interface ExplainPlaceholdersChildProps extends FetchExplainsResult {
  databaseType: DatabasesType;
  example?: QueryExampleResponseItem;
}

export interface ExplainPlaceholdersProps {
  queryId?: string;
  databaseType: DatabasesType;
  examples: QueryExampleResponseItem[];
  children: (result: ExplainPlaceholdersChildProps) => ReactNode;
}
