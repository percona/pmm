import { RadioGroup } from '@percona/peak-ui';
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
import {
  AUTO_DISCOVERY_OPTIONS,
  MAX_CONNECTIONS_OPTIONS,
  POSTGRESQL_DEFAULTS,
  TRACKING_OPTIONS,
} from './PostgreSQLForm.constants';
import { Messages } from './PostgreSQLForm.messages';
import { postgresqlSchema } from './PostgreSQLForm.schema';
import {
  AutoDiscoveryOption,
  MaxConnectionsOption,
} from './PostgreSQLForm.types';
import { toPostgreSQLPayload } from './PostgreSQLForm.utils';

export const PostgreSQLForm: FC = () => {
  const form = useServiceForm(postgresqlSchema, POSTGRESQL_DEFAULTS);
  const { nodes } = useNodeOptions();
  const { mutateAsync: addService } = useAddService();

  return (
    <ServiceForm
      form={form}
      serviceTypeLabel={PageMessages.serviceTypes.postgresql}
      onSubmit={(values) => addService(toPostgreSQLPayload(values, nodes))}
    >
      <ConnectionDetails withNodes>
        <TextField
          name="database"
          label={Messages.database.label}
          placeholder={Messages.database.placeholder}
          tooltip={Messages.database.tooltip}
        />
        <MaxQueryLengthField />
      </ConnectionDetails>
      <Labels />
      <AdditionalOptions>
        <TlsOptions certificates={<TlsCertificates />} />
        <QueryAnalyticsOptions />
        <DisableCollectorsField />
        <RadioGroup
          name="tracking"
          label={Messages.tracking.label}
          options={TRACKING_OPTIONS}
        />
        <OptionWithLimit
          name="autoDiscoveryOptions"
          label={Messages.autoDiscovery.label}
          options={AUTO_DISCOVERY_OPTIONS}
          editableOption={AutoDiscoveryOption.custom}
          limitName="autoDiscoveryLimit"
          limitLabel={Messages.autoDiscovery.limit}
          limitTooltip={Messages.autoDiscovery.tooltip}
        />
        <OptionWithLimit
          name="maxConnectionsOptions"
          label={Messages.maxConnections.label}
          options={MAX_CONNECTIONS_OPTIONS}
          editableOption={MaxConnectionsOption.enabled}
          limitName="maxExporterConnections"
          limitLabel={Messages.maxConnections.limit}
          limitTooltip={Messages.maxConnections.tooltip}
        />
      </AdditionalOptions>
    </ServiceForm>
  );
};
