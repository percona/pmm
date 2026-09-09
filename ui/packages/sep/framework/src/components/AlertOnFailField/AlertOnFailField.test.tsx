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

import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {
  useForm,
  FormProvider,
  type FieldValues,
  type UseFormGetValues,
} from 'react-hook-form';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';

const useAlertConfigMock = vi.fn();

vi.mock('@sep/api', () => ({
  useAlertConfig: () => useAlertConfigMock(),
}));

import { AlertOnFailField, ALERT_ON_FAIL_FIELD_NAME } from './AlertOnFailField';

type AlertConfigState = {
  data?: { available: boolean };
  isLoading: boolean;
  isError?: boolean;
};

function setAlertConfig(state: AlertConfigState) {
  useAlertConfigMock.mockReturnValue({ isError: false, ...state });
}

function makeQueryClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0, staleTime: 0 } },
  });
}

function Harness({
  defaultValue,
  onSubmit,
  formSpy,
}: {
  defaultValue?: boolean;
  onSubmit?: (values: Record<string, unknown>) => void;
  formSpy?: (api: { getValues: UseFormGetValues<FieldValues> }) => void;
}) {
  const methods = useForm();
  formSpy?.({ getValues: methods.getValues });
  return (
    <FormProvider {...methods}>
      <form onSubmit={methods.handleSubmit((v) => onSubmit?.(v))}>
        <AlertOnFailField defaultValue={defaultValue} />
        <button type="submit">submit</button>
      </form>
    </FormProvider>
  );
}

function renderHarness(ui: ReactNode) {
  return render(
    <QueryClientProvider client={makeQueryClient()}>{ui}</QueryClientProvider>
  );
}

beforeEach(() => {
  useAlertConfigMock.mockReset();
});

describe('AlertOnFailField', () => {
  it('uses the snake_case form field name expected by the API', () => {
    expect(ALERT_ON_FAIL_FIELD_NAME).toBe('alert_on_fail');
  });

  it('renders an enabled checkbox when providers are configured', () => {
    setAlertConfig({ data: { available: true }, isLoading: false });
    renderHarness(<Harness />);

    const checkbox = screen.getByRole('checkbox', {
      name: /Alert on failure/i,
    });
    expect(checkbox).not.toBeDisabled();
    expect(checkbox).not.toBeChecked();
  });

  it('renders a disabled checkbox when no providers are configured', () => {
    setAlertConfig({ data: { available: false }, isLoading: false });
    renderHarness(<Harness />);

    const checkbox = screen.getByRole('checkbox', {
      name: /Alert on failure/i,
    });
    expect(checkbox).toBeDisabled();
    expect(checkbox).not.toBeChecked();
  });

  it('keeps the checkbox disabled while the availability query is loading', () => {
    setAlertConfig({ data: undefined, isLoading: true });
    renderHarness(<Harness />);

    const checkbox = screen.getByRole('checkbox', {
      name: /Alert on failure/i,
    });
    expect(checkbox).toBeDisabled();
  });

  it('disables the checkbox when the availability query errors', () => {
    setAlertConfig({ data: undefined, isLoading: false, isError: true });
    renderHarness(<Harness />);

    const checkbox = screen.getByRole('checkbox', {
      name: /Alert on failure/i,
    });
    expect(checkbox).toBeDisabled();
  });

  it('honors defaultValue=true when providers are available', () => {
    setAlertConfig({ data: { available: true }, isLoading: false });
    renderHarness(<Harness defaultValue />);

    const checkbox = screen.getByRole('checkbox', {
      name: /Alert on failure/i,
    });
    expect(checkbox).toBeChecked();
  });

  it('clears defaultValue=true when providers are unavailable at mount', async () => {
    setAlertConfig({ data: { available: false }, isLoading: false });
    let getValues: UseFormGetValues<FieldValues> | undefined;
    renderHarness(
      <Harness
        defaultValue
        formSpy={({ getValues: gv }) => {
          getValues = gv;
        }}
      />
    );

    const checkbox = screen.getByRole('checkbox', {
      name: /Alert on failure/i,
    });
    expect(checkbox).toBeDisabled();
    await waitFor(() => expect(checkbox).not.toBeChecked());
    // Form state, not just rendered checkbox, must be cleared.
    expect(getValues?.('alert_on_fail')).toBe(false);
  });

  it('clears the value when providers become unavailable mid-session', async () => {
    setAlertConfig({ data: { available: true }, isLoading: false });
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    let getValues: UseFormGetValues<FieldValues> | undefined;
    const captureSpy = ({
      getValues: gv,
    }: {
      getValues: UseFormGetValues<FieldValues>;
    }) => {
      getValues = gv;
    };
    const { rerender } = renderHarness(
      <Harness onSubmit={onSubmit} formSpy={captureSpy} />
    );

    const checkbox = screen.getByRole('checkbox', {
      name: /Alert on failure/i,
    });
    await user.click(checkbox);
    expect(checkbox).toBeChecked();

    setAlertConfig({ data: { available: false }, isLoading: false });
    rerender(
      <QueryClientProvider client={makeQueryClient()}>
        <Harness onSubmit={onSubmit} formSpy={captureSpy} />
      </QueryClientProvider>
    );

    await waitFor(() => expect(checkbox).not.toBeChecked());
    expect(checkbox).toBeDisabled();
    expect(getValues?.('alert_on_fail')).toBe(false);

    // Submitting also produces the cleared value — proves the reset is in
    // form state, not just a render-time mask.
    await user.click(screen.getByRole('button', { name: 'submit' }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalled());
    expect(onSubmit).toHaveBeenLastCalledWith(
      expect.objectContaining({ alert_on_fail: false })
    );
  });

  it('keeps defaultValue=true after a delayed available=true query resolution', async () => {
    setAlertConfig({ data: undefined, isLoading: true });
    const { rerender } = renderHarness(<Harness defaultValue />);

    const checkbox = screen.getByRole('checkbox', {
      name: /Alert on failure/i,
    });
    // While loading, the field is disabled but the initial form value is preserved.
    expect(checkbox).toBeDisabled();

    setAlertConfig({ data: { available: true }, isLoading: false });
    rerender(
      <QueryClientProvider client={makeQueryClient()}>
        <Harness defaultValue />
      </QueryClientProvider>
    );

    await waitFor(() => expect(checkbox).not.toBeDisabled());
    expect(checkbox).toBeChecked();
  });

  it('submits the toggled value under the alert_on_fail key', async () => {
    setAlertConfig({ data: { available: true }, isLoading: false });
    const onSubmit = vi.fn();
    const user = userEvent.setup();
    renderHarness(<Harness onSubmit={onSubmit} />);

    const checkbox = screen.getByRole('checkbox', {
      name: /Alert on failure/i,
    });
    await user.click(checkbox);
    await user.click(screen.getByRole('button', { name: 'submit' }));

    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ alert_on_fail: true })
    );
  });
});
