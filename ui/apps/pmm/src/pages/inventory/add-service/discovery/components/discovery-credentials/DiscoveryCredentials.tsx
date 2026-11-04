import Stack from '@mui/material/Stack';
import { PMM_ADD_SERVICE_PATH } from 'lib/constants';
import { FormProvider, type FieldValues } from 'react-hook-form';
import { AddServiceActions } from '../../../add-service-actions';
import { Messages } from './DiscoveryCredentials.messages';
import type { DiscoveryCredentialsProps } from './DiscoveryCredentials.types';

export const DiscoveryCredentials = <T extends FieldValues>({
  form,
  onSubmit,
  isDiscovering,
  testId,
  children,
}: DiscoveryCredentialsProps<T>) => (
  <FormProvider {...form}>
    <Stack
      component="form"
      data-testid={testId}
      noValidate
      gap={3}
      onSubmit={form.handleSubmit(onSubmit)}
    >
      {children}
      <AddServiceActions
        cancelTo={PMM_ADD_SERVICE_PATH}
        submitLabel={Messages.discover}
        isSubmitting={isDiscovering}
      />
    </Stack>
  </FormProvider>
);
