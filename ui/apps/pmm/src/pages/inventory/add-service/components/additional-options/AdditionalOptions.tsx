import Stack from '@mui/material/Stack';
import type { FC, PropsWithChildren } from 'react';
import { SwitchInput } from '@percona/peak-ui';
import { FormSection } from '../form-section';
import { Messages } from './AdditionalOptions.messages';

export const AdditionalOptions: FC<PropsWithChildren> = ({ children }) => (
  <FormSection title={Messages.title}>
    <Stack sx={{ mt: 1 }}>
      <SwitchInput
        name="skipConnectionCheck"
        label={Messages.skipConnectionCheck}
      />
      {children}
    </Stack>
  </FormSection>
);
