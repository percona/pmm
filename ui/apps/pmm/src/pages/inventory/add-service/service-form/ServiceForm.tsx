import Stack from '@mui/material/Stack';
import { isAxiosError } from 'axios';
import {
  PMM_ADD_SERVICE_PATH,
  PMM_INVENTORY_SERVICES_PATH,
} from 'lib/constants';
import { enqueueSnackbar } from 'notistack';
import { FormProvider, type FieldValues } from 'react-hook-form';
import { useNavigate } from 'react-router-dom';
import { AddServiceActions } from '../add-service-actions';
import { Messages } from './ServiceForm.messages';
import type {
  ConnectionFormValues,
  ServiceFormProps,
} from './ServiceForm.types';
import { getErrorMessage, getServiceName } from './ServiceForm.utils';

export const ServiceForm = <
  T extends FieldValues & Pick<ConnectionFormValues, 'serviceName' | 'address'>,
>({
  form,
  serviceTypeLabel,
  onSubmit,
  onCancel,
  children,
}: ServiceFormProps<T>) => {
  const navigate = useNavigate();

  const handleSubmit = async (values: T) => {
    try {
      await onSubmit(values);
    } catch (error) {
      // Keep the form for a retry; the API layer has usually reported the failure already.
      if (!isAxiosError(error) || !error.notified) {
        enqueueSnackbar(getErrorMessage(error), { variant: 'error' });
      }
      return;
    }

    enqueueSnackbar(
      Messages.success(getServiceName(values), serviceTypeLabel),
      { variant: 'success' }
    );
    navigate(PMM_INVENTORY_SERVICES_PATH);
  };

  return (
    <FormProvider {...form}>
      <Stack
        component="form"
        data-testid="add-service-form"
        noValidate
        gap={4}
        onSubmit={form.handleSubmit(handleSubmit)}
      >
        {children}
        <AddServiceActions
          {...(onCancel ? { onCancel } : { cancelTo: PMM_ADD_SERVICE_PATH })}
          submitLabel={Messages.addService}
          isSubmitting={form.formState.isSubmitting}
          isSticky
        />
      </Stack>
    </FormProvider>
  );
};
