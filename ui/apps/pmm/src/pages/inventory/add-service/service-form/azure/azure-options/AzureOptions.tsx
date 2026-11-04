import { SwitchInput } from '@percona/peak-ui';
import type { FC } from 'react';
import { Messages } from '../AzureServiceForm.messages';

export const AzureOptions: FC = () => (
  <SwitchInput
    name="azureDatabaseExporter"
    label={Messages.azureDatabaseExporter}
  />
);
