import type { FC } from 'react';
import { DiscoverAzureDatabaseType } from 'types/services.types';
import { AzureMySQLForm } from './azure-mysql';
import { AzurePostgreSQLForm } from './azure-postgresql';
import type { AzureServiceFormProps } from './AzureServiceForm.types';

export const AzureServiceForm: FC<AzureServiceFormProps> = (props) =>
  props.instance.type === DiscoverAzureDatabaseType.postgresql ? (
    <AzurePostgreSQLForm {...props} />
  ) : (
    <AzureMySQLForm {...props} />
  );
