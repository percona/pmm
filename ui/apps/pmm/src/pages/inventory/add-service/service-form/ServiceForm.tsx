import Stack from '@mui/material/Stack';
import { isAxiosError } from 'axios';
import { PMM_ADD_SERVICE_PATH, PMM_INVENTORY_URL } from 'lib/constants';
import { enqueueSnackbar } from 'notistack';
import { FormProvider, type FieldValues } from 'react-hook-form';
import { useNavigate } from 'react-router-dom';
import { AddServiceActions } from '../add-service-actions';
import { Messages } from './ServiceForm.messages';
import type {
  ConnectionFormValues,
  ServiceFormProps,
} from './ServiceForm.types';
import { getServiceName } from './ServiceForm.utils';

export const ServiceForm = <
  T extends FieldValues & Pick<ConnectionFormValues, 'serviceName' | 'address'>,
>({
  form,
  serviceTypeLabel,
  onSubmit,
  children,
}: ServiceFormProps<T>) => {
  const navigate = useNavigate();

  const handleSubmit = async (values: T) => {
    try {
      await onSubmit(values);
    } catch (error) {
      // The API layer reports failed responses itself; keep the form for a retry.
      if (!isAxiosError(error) || !error.response) {
        enqueueSnackbar(
          error instanceof Error ? error.message : Messages.unknownError,
          { variant: 'error' }
        );
      }
      return;
    }

    enqueueSnackbar(
      Messages.success(getServiceName(values), serviceTypeLabel),
      { variant: 'success' }
    );
    navigate(PMM_INVENTORY_URL);
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
          cancelTo={PMM_ADD_SERVICE_PATH}
          submitLabel={Messages.addService}
          isSubmitting={form.formState.isSubmitting}
        />
      </Stack>
    </FormProvider>
  );
};
