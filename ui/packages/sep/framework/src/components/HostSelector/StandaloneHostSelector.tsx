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
import { Autocomplete, TextField } from '@mui/material';
import type { SxProps } from '@mui/material/styles';
import { useSnackbar } from 'notistack';
import { useHosts, type HostOption } from '../../hooks/useHosts';

const EMPTY_OPTIONS: HostOption[] = [];

type HostIdOption = { id: string; label: string };

export interface StandaloneHostSelectorProps {
  value: string;
  onChange: (hostId: string) => void;
  label?: string;
  disabled?: boolean;
  sx?: SxProps;
}

export function StandaloneHostSelector({
  value,
  onChange,
  label = 'Executor Host',
  disabled,
  sx,
}: StandaloneHostSelectorProps) {
  const { enqueueSnackbar } = useSnackbar();
  const { data, isLoading, isError, error, refetch } = useHosts();
  const hosts = data ?? EMPTY_OPTIONS;

  // Surface a hosts-query failure (e.g. an upstream Tasks-API 502) via the
  // shell's snackbar. De-dup on the `error` object identity (stable per
  // failure) so the snackbar fires once per failure, not once per render.
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

  const options: HostIdOption[] = hosts.map((h) => ({
    id: h.id,
    label: h.name,
  }));
  const selected = options.find((o) => o.id === value) ?? null;

  return (
    <Autocomplete<HostIdOption>
      data-testid="host-selector"
      options={options}
      value={selected}
      onChange={(_, opt) => onChange(opt?.id ?? '')}
      onOpen={() => refetch()}
      getOptionLabel={(o) => o.label}
      isOptionEqualToValue={(a, b) => a.id === b.id}
      loading={isLoading}
      loadingText="Loading hosts…"
      noOptionsText="No hosts available"
      disabled={disabled || isError}
      renderInput={(params) => (
        <TextField
          {...params}
          label={label}
          error={isError}
          helperText={
            isError ? (error?.message ?? 'Failed to load hosts') : undefined
          }
        />
      )}
      sx={sx}
    />
  );
}
