import { SwitchInput } from '@percona/peak-ui';
import type { FC } from 'react';
import { AddServiceType } from '../../../AddServicePage.types';
import { DisableCollectorsField } from '../../../components/disable-collectors';
import { TextField } from '../../../components/fields';
import { OptionWithLimit } from '../../../components/option-with-limit';
import { QueryAnalyticsOptions } from '../../../components/query-analytics-options';
import { TlsCertificates } from '../../../components/tls-certificates';
import { TlsOptions } from '../../../components/tls-options';
import { TABLESTAT_LIMITS, TABLESTAT_OPTIONS } from '../MySQLForm.constants';
import { Messages } from '../MySQLForm.messages';
import { TablestatOption } from '../MySQLForm.types';
import type { MySQLOptionsProps } from './MySQLOptions.types';

export const MySQLOptions: FC<MySQLOptionsProps> = ({ cloud }) => {
  const isAzure = cloud === AddServiceType.azure;

  return (
    <>
      <TlsOptions certificates={!cloud && <TlsCertificates />} />
      <QueryAnalyticsOptions withCommentsParsing={!isAzure} />
      <SwitchInput
        name="qanMysqlPerfschema"
        label={Messages.qanMysqlPerfschema}
      />
      {!cloud && (
        <TextField
          name="extraDsnParams"
          kind="textarea"
          label={Messages.extraDsnParams.label}
          placeholder={Messages.extraDsnParams.placeholder}
          helperText={Messages.extraDsnParams.description}
        />
      )}
      {!isAzure && <DisableCollectorsField />}
      <OptionWithLimit
        name="tablestatOptions"
        label={Messages.tablestats.label}
        options={TABLESTAT_OPTIONS}
        editableOption={TablestatOption.custom}
        limitName="tablestatsGroupTableLimit"
        limitLabel={Messages.tablestats.limit}
        presets={TABLESTAT_LIMITS}
      />
    </>
  );
};
