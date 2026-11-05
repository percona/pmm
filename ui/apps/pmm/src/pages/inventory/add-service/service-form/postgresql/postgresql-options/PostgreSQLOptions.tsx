import { RadioGroup } from '@percona/peak-ui';
import type { FC } from 'react';
import { DisableCollectorsField } from '../../../components/disable-collectors';
import { OptionWithLimit } from '../../../components/option-with-limit';
import { QueryAnalyticsOptions } from '../../../components/query-analytics-options';
import { TlsCertificates } from '../../../components/tls-certificates';
import { TlsOptions } from '../../../components/tls-options';
import {
  AUTO_DISCOVERY_LIMITS,
  AUTO_DISCOVERY_OPTIONS,
  MAX_CONNECTIONS_OPTIONS,
  TRACKING_OPTIONS,
} from '../PostgreSQLForm.constants';
import { Messages } from '../PostgreSQLForm.messages';
import {
  AutoDiscoveryOption,
  MaxConnectionsOption,
} from '../PostgreSQLForm.types';
import type { PostgreSQLOptionsProps } from './PostgreSQLOptions.types';

export const PostgreSQLOptions: FC<PostgreSQLOptionsProps> = ({
  trackingOptions = TRACKING_OPTIONS,
  withExporterOptions = true,
}) => (
  <>
    <TlsOptions certificates={<TlsCertificates />} />
    <QueryAnalyticsOptions />
    {withExporterOptions && <DisableCollectorsField />}
    <RadioGroup
      name="tracking"
      label={Messages.tracking.label}
      options={trackingOptions}
    />
    <OptionWithLimit
      name="autoDiscoveryOptions"
      label={Messages.autoDiscovery.label}
      options={AUTO_DISCOVERY_OPTIONS}
      editableOption={AutoDiscoveryOption.custom}
      limitName="autoDiscoveryLimit"
      limitLabel={Messages.autoDiscovery.limit}
      presets={AUTO_DISCOVERY_LIMITS}
      limitTooltip={Messages.autoDiscovery.tooltip}
    />
    {withExporterOptions && (
      <OptionWithLimit
        name="maxConnectionsOptions"
        label={Messages.maxConnections.label}
        options={MAX_CONNECTIONS_OPTIONS}
        editableOption={MaxConnectionsOption.enabled}
        limitName="maxExporterConnections"
        limitLabel={Messages.maxConnections.limit}
        limitTooltip={Messages.maxConnections.tooltip}
      />
    )}
  </>
);
