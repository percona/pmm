import { QueryData } from 'types/rta.types';
import { CodeLanguage } from 'types/util.types';

/**
 * Returns the syntax-highlighting language for a query based on its
 * service-specific payload. MySQL queries are plain SQL while MongoDB
 * queries use the mongodb (JSON-like) syntax.
 */
export const getQueryLanguage = (queryData: QueryData): CodeLanguage =>
  queryData.mySqlPayload ? 'sql' : 'mongodb';
