import { SwitchInput } from '@percona/peak-ui';
import type { FC } from 'react';
import { Messages } from '../RdsServiceForm.messages';

export const RdsOptions: FC = () => (
  <>
    <SwitchInput
      name="disableBasicMetrics"
      label={Messages.disableBasicMetrics}
    />
    <SwitchInput
      name="disableEnhancedMetrics"
      label={Messages.disableEnhancedMetrics}
    />
  </>
);
