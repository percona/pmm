import { SwitchInput } from '@percona/peak-ui';
import type { FC } from 'react';
import { useFormContext, useWatch } from 'react-hook-form';
import type { TlsFormValues } from '../../service-form/ServiceForm.types';
import { TlsSkipVerify } from '../tls-skip-verify';
import { Messages } from './TlsOptions.messages';
import type { TlsOptionsProps } from './TlsOptions.types';

export const TlsOptions: FC<TlsOptionsProps> = ({ certificates }) => {
  const { control } = useFormContext<TlsFormValues>();
  const tls = useWatch({ control, name: 'tls' });

  return (
    <>
      <SwitchInput name="tls" label={Messages.tls} />
      {tls && certificates}
      <TlsSkipVerify />
    </>
  );
};
