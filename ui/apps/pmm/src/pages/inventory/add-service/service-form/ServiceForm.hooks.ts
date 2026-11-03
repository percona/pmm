import { zodResolver } from '@hookform/resolvers/zod';
import { type DefaultValues, type FieldValues, useForm } from 'react-hook-form';
import type { z } from 'zod';

export const useServiceForm = <T extends FieldValues>(
  schema: z.ZodType<T, T>,
  defaultValues: DefaultValues<T>
) =>
  useForm<T>({
    resolver: zodResolver(schema),
    // Errors show once a field is left, as in the old form.
    mode: 'onTouched',
    defaultValues,
  });
