import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import {
  RadioGroup,
  SelectInput,
  SwitchInput,
  TextInput,
} from '@percona/peak-ui';
import type { FC } from 'react';
import { type ManagedServiceType, MetricsMode } from 'types/services.types';
import { getLogLevels } from '../EditService.constants';
import { Messages } from '../EditService.messages';
import type { EditServiceFormValues } from '../EditService.schema';
import type { SettingName } from '../EditService.types';
import { SectionHeader } from '../section-header';
import { useIsSettingShown } from '../useIsSettingShown';

const { metrics } = Messages;

const METRICS_MODE_OPTIONS = [
  { value: MetricsMode.pull, label: metrics.pull },
  { value: MetricsMode.push, label: metrics.push },
];

// Text and number settings, in the order the form shows them
const FIELDS: [SettingName, string, string, boolean][] = [
  [
    'disableCollectors',
    metrics.disableCollectors,
    metrics.disableCollectorsDescription,
    false,
  ],
  [
    'tablestatsGroupTableLimit',
    metrics.tablestatsGroupTableLimit,
    metrics.tablestatsGroupTableLimitDescription,
    true,
  ],
  [
    'statsCollections',
    metrics.statsCollections,
    metrics.statsCollectionsDescription,
    false,
  ],
  [
    'collectionsLimit',
    metrics.collectionsLimit,
    metrics.collectionsLimitDescription,
    true,
  ],
  [
    'autoDiscoveryLimit',
    metrics.autoDiscoveryLimit,
    metrics.autoDiscoveryLimitDescription,
    true,
  ],
  [
    'maxExporterConnections',
    metrics.maxExporterConnections,
    metrics.maxExporterConnectionsDescription,
    true,
  ],
];

const SWITCHES: [SettingName, string][] = [
  ['exposeExporter', metrics.exposeExporter],
  ['enableAllCollectors', metrics.enableAllCollectors],
  ['enableDiagnosticDataHistograms', metrics.enableDiagnosticDataHistograms],
];

interface Props {
  serviceType: ManagedServiceType;
}

export const MetricsSettings: FC<Props> = ({ serviceType }) => {
  const isShown = useIsSettingShown(serviceType);
  const switches = SWITCHES.filter(([name]) => isShown(name));

  return (
    <Stack gap={2} data-testid="metrics-settings">
      <SectionHeader title={metrics.title} />
      <RadioGroup<EditServiceFormValues>
        name="metricsMode"
        label={metrics.metricsMode}
        options={METRICS_MODE_OPTIONS}
      />
      {isShown('logLevel') && (
        <Stack direction={{ xs: 'column', md: 'row' }} gap={2}>
          <SelectInput<EditServiceFormValues>
            name="logLevel"
            label={metrics.logLevel}
            formControlProps={{ fullWidth: true }}
          >
            {getLogLevels(serviceType).map((level) => (
              <MenuItem key={level} value={level}>
                {metrics.logLevels[level]}
              </MenuItem>
            ))}
          </SelectInput>
          <TextInput<EditServiceFormValues>
            name="connectionTimeout"
            label={metrics.connectionTimeout}
            textFieldProps={{
              type: 'number',
              fullWidth: true,
              helperText: metrics.connectionTimeoutDescription,
            }}
          />
        </Stack>
      )}
      {FIELDS.filter(([name]) => isShown(name)).map(
        ([name, label, helperText, isNumber]) => (
          <TextInput<EditServiceFormValues>
            key={name}
            name={name}
            label={label}
            textFieldProps={{
              type: isNumber ? 'number' : 'text',
              fullWidth: true,
              helperText,
            }}
          />
        )
      )}
      {!!switches.length && (
        <Stack>
          {switches.map(([name, label]) => (
            <SwitchInput<EditServiceFormValues>
              key={name}
              name={name}
              label={label}
            />
          ))}
        </Stack>
      )}
    </Stack>
  );
};
