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

import { fireEvent, render, screen, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ConfigForm } from '../src/components/ConfigForm';
import type { OmInventorySetting } from '../src/types';

const {
  useOmInventoryConfig,
  useUpdateOmInventoryConfig,
  useResetOmInventoryConfig,
  enqueueSnackbar,
} = vi.hoisted(() => ({
  enqueueSnackbar: vi.fn(),
  useOmInventoryConfig: vi.fn(),
  useUpdateOmInventoryConfig: vi.fn(),
  useResetOmInventoryConfig: vi.fn(),
}));

vi.mock('notistack', () => ({ enqueueSnackbar }));
vi.mock('../src/inventoryHooks', () => ({
  useOmInventoryConfig,
  useUpdateOmInventoryConfig,
  useResetOmInventoryConfig,
}));

const setting = (
  overrides: Partial<OmInventorySetting> & Pick<OmInventorySetting, 'key'>
): OmInventorySetting => ({
  value: 4,
  default_value: 4,
  type: 'int',
  reload: 'SETTING_RELOAD_HOT',
  has_override: false,
  is_advanced: false,
  ...overrides,
});

const SETTINGS = [
  setting({ key: 'SCHEDULE__every', value: 30 }),
  setting({ key: 'MAX_CONCURRENT_PROBES', has_override: true }),
];

const idle = { mutate: vi.fn(), isPending: false, isError: false };

describe('ConfigForm', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useOmInventoryConfig.mockReturnValue({
      data: SETTINGS,
      isPending: false,
      isError: false,
    });
    useUpdateOmInventoryConfig.mockReturnValue(idle);
    useResetOmInventoryConfig.mockReturnValue(idle);
  });

  it('says a save landed', () => {
    const mutate = vi.fn(
      (_values: unknown, options?: { onSuccess?: () => void }) =>
        options?.onSuccess?.()
    );
    useUpdateOmInventoryConfig.mockReturnValue({ ...idle, mutate });
    render(<ConfigForm group="advanced" />);

    fireEvent.change(screen.getByLabelText(/^Concurrent scans/), {
      target: { value: '8' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    expect(enqueueSnackbar).toHaveBeenCalledWith('Saved', {
      variant: 'success',
    });
  });

  it('reports a failed save', () => {
    useUpdateOmInventoryConfig.mockReturnValue({
      ...idle,
      isError: true,
      error: new Error('CONNECT_TIMEOUT: must be positive'),
    });
    render(<ConfigForm group="general" />);

    expect(screen.getByRole('alert')).toHaveTextContent(
      'CONNECT_TIMEOUT: must be positive'
    );
  });

  it('reports a failed reset', () => {
    useResetOmInventoryConfig.mockReturnValue({
      ...idle,
      isError: true,
      error: new Error('reset refused'),
    });
    render(<ConfigForm group="advanced" />);

    expect(screen.getByText('reset refused')).toBeInTheDocument();
  });

  it('names an invalid field on another tab by its label and tab', () => {
    const { rerender } = render(<ConfigForm group="advanced" />);
    fireEvent.change(screen.getByLabelText(/^Concurrent scans/), {
      target: { value: 'abc' },
    });
    rerender(<ConfigForm group="general" />);

    expect(
      screen.getByText('Concurrent scans (Advanced tab)')
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled();
  });

  it('lists a pending change by its label', () => {
    render(<ConfigForm group="general" />);
    fireEvent.change(screen.getByLabelText(/^Scan every/), {
      target: { value: '60' },
    });

    const saveRow = screen.getByRole('button', { name: 'Save' })
      .parentElement as HTMLElement;
    expect(within(saveRow).getByText('Scan every')).toBeInTheDocument();
    expect(screen.queryByText('SCHEDULE__every')).toBeNull();
  });
});
