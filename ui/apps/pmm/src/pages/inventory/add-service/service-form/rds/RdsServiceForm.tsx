import type { FC } from 'react';
import { DiscoverRDSEngine } from 'types/services.types';
import { RdsMySQLForm } from './rds-mysql';
import { RdsPostgreSQLForm } from './rds-postgresql';
import type { RdsServiceFormProps } from './RdsServiceForm.types';

export const RdsServiceForm: FC<RdsServiceFormProps> = (props) =>
  props.instance.engine === DiscoverRDSEngine.postgresql ? (
    <RdsPostgreSQLForm {...props} />
  ) : (
    <RdsMySQLForm {...props} />
  );
