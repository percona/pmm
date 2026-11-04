import { SwitchInput } from '@percona/peak-ui';
import type { FC } from 'react';
import { DisableCollectorsField } from '../../../components/disable-collectors';
import { TextField } from '../../../components/fields';
import { OptionWithLimit } from '../../../components/option-with-limit';
import { QueryAnalyticsOptions } from '../../../components/query-analytics-options';
import { TlsCertificates } from '../../../components/tls-certificates';
import { TlsOptions } from '../../../components/tls-options';
import { TABLESTAT_OPTIONS } from '../MySQLForm.constants';
import { Messages } from '../MySQLForm.messages';
import { TablestatOption } from '../MySQLForm.types';
import type { MySQLOptionsProps } from './MySQLOptions.types';

export const MySQLOptions: FC<MySQLOptionsProps> = ({
  withCollectors = true,
}) => (
  <>
    <TlsOptions certificates={<TlsCertificates />} />
    <QueryAnalyticsOptions />
    <SwitchInput
      name="qanMysqlPerfschema"
      label={Messages.qanMysqlPerfschema}
    />
    <TextField
      name="extraDsnParams"
      kind="textarea"
      label={Messages.extraDsnParams.label}
      placeholder={Messages.extraDsnParams.placeholder}
      helperText={Messages.extraDsnParams.description}
    />
    {withCollectors && <DisableCollectorsField />}
    <OptionWithLimit
      name="tablestatOptions"
      label={Messages.tablestats.label}
      options={TABLESTAT_OPTIONS}
      editableOption={TablestatOption.custom}
      limitName="tablestatsGroupTableLimit"
      limitLabel={Messages.tablestats.limit}
    />
  </>
);
