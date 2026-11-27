import { SwitchInput } from '@percona/peak-ui';
import type { FC } from 'react';
import { Messages } from './TlsSkipVerify.messages';

export const TlsSkipVerify: FC = () => (
  <SwitchInput name="tlsSkipVerify" label={Messages.label} />
);
