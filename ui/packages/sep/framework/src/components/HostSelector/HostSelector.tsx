/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

import { useEffect, useRef } from 'react';
import { useFormContext } from 'react-hook-form';
import { AutoCompleteInput } from '@percona/percona-ui';
import { useSnackbar } from 'notistack';
import { useHosts, type HostOption } from '../../hooks/useHosts';

const EMPTY_OPTIONS: HostOption[] = [];

export interface HostSelectorProps {
  /** react-hook-form field name. Stores a `HostOption | null`. */
  name: string;
  label: string;
  required?: boolean;
  disabled?: boolean;
  helperText?: string;
}

const getOptionLabel = (opt: HostOption | string) =>
  typeof opt === 'string' ? opt : opt.name;

const isOptionEqualToValue = (a: HostOption, b: HostOption) => a.id === b.id;

export function HostSelector({
  name,
  label,
  required,
  disabled,
  helperText,
}: HostSelectorProps) {
  const {
    control,
    formState: { errors },
  } = useFormContext();
  const { enqueueSnackbar } = useSnackbar();

  const { data, isLoading, isError, error, refetch } = useHosts();
  const hosts = data ?? EMPTY_OPTIONS;

  const empty = !isLoading && !isError && hosts.length === 0;

  const fieldError = errors[name]?.message as string | undefined;

  // Surface a hosts-query failure (e.g. an upstream Tasks-API 502) via the
  // shell's snackbar. React Query keeps the `error` object identity stable
  // between renders until the next refetch, so de-dup on that identity to
  // raise the snackbar once per failure rather than once per render.
  const lastSurfacedRef = useRef<unknown>(null);
  useEffect(() => {
    if (isError && error && error !== lastSurfacedRef.current) {
      enqueueSnackbar(`Failed to load executor hosts: ${error.message}`, {
        variant: 'error',
        autoHideDuration: 30_000,
      });
      lastSurfacedRef.current = error;
    }
  }, [isError, error, enqueueSnackbar]);

  let text = helperText;
  if (fieldError) {
    text = fieldError;
  } else if (isError) {
    text = error?.message ?? 'Failed to load hosts';
  } else if (empty) {
    text = 'No hosts available';
  }

  return (
    <AutoCompleteInput<HostOption>
      name={name}
      label={label}
      control={control}
      isRequired={required}
      loading={isLoading}
      disabled={disabled || isError}
      options={hosts}
      controllerProps={{
        rules: required ? { required: `${label} is required` } : undefined,
      }}
      autoCompleteProps={{
        getOptionLabel,
        isOptionEqualToValue,
        noOptionsText: isLoading ? 'Loading hosts…' : 'No hosts available',
        onOpen: () => refetch(),
      }}
      textFieldProps={{ helperText: text, error: isError || !!fieldError }}
    />
  );
}
