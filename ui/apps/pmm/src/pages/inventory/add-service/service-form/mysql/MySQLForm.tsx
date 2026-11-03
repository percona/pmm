import { SwitchInput } from '@percona/peak-ui';
import { useAddService } from 'hooks/api/useServices';
import type { FC } from 'react';
import { Messages as PageMessages } from '../../AddServicePage.messages';
import { AdditionalOptions } from '../../components/additional-options';
import { ConnectionDetails } from '../../components/connection-details';
import { DisableCollectorsField } from '../../components/disable-collectors';
import { TextField } from '../../components/fields';
import { Labels } from '../../components/labels';
import { MaxQueryLengthField } from '../../components/max-query-length';
import { useNodeOptions } from '../../components/nodes-agents';
import { OptionWithLimit } from '../../components/option-with-limit';
import { QueryAnalyticsOptions } from '../../components/query-analytics-options';
import { TlsCertificates } from '../../components/tls-certificates';
import { TlsOptions } from '../../components/tls-options';
import { ServiceForm } from '../ServiceForm';
import { useServiceForm } from '../ServiceForm.hooks';
import { MYSQL_DEFAULTS, TABLESTAT_OPTIONS } from './MySQLForm.constants';
import { Messages } from './MySQLForm.messages';
import { mysqlSchema } from './MySQLForm.schema';
import { TablestatOption } from './MySQLForm.types';
import { toMySQLPayload } from './MySQLForm.utils';

export const MySQLForm: FC = () => {
  const form = useServiceForm(mysqlSchema, MYSQL_DEFAULTS);
  const { nodes } = useNodeOptions();
  const { mutateAsync: addService } = useAddService();

  return (
    <ServiceForm
      form={form}
      serviceTypeLabel={PageMessages.serviceTypes.mysql}
      onSubmit={(values) => addService(toMySQLPayload(values, nodes))}
    >
      <ConnectionDetails withNodes>
        <MaxQueryLengthField />
      </ConnectionDetails>
      <Labels />
      <AdditionalOptions>
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
        <DisableCollectorsField />
        <OptionWithLimit
          name="tablestatOptions"
          label={Messages.tablestats.label}
          options={TABLESTAT_OPTIONS}
          editableOption={TablestatOption.custom}
          limitName="tablestatsGroupTableLimit"
          limitLabel={Messages.tablestats.limit}
        />
      </AdditionalOptions>
    </ServiceForm>
  );
};
