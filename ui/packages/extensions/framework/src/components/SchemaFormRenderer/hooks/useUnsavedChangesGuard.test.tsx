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

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useUnsavedChangesGuard } from './useUnsavedChangesGuard';

// ── Mock setup ────────────────────────────────────────────────────────────────

const mockReset = vi.hoisted(() => vi.fn());
const mockUseFormContext = vi.hoisted(() => vi.fn());

vi.mock('react-hook-form', async (importOriginal) => {
  const actual = (await importOriginal()) as Record<string, unknown>;
  return {
    ...actual,
    useFormContext: mockUseFormContext,
  };
});

function setFormState({
  isDirty = false,
  isSubmitSuccessful = false,
}: {
  isDirty?: boolean;
  isSubmitSuccessful?: boolean;
}) {
  mockUseFormContext.mockReturnValue({
    formState: { isDirty, isSubmitSuccessful },
    reset: mockReset,
  });
}

// ── Predicate tests ───────────────────────────────────────────────────────────

describe('useUnsavedChangesGuard — isGuarded predicate', () => {
  beforeEach(() => {
    mockReset.mockReset();
    setFormState({});
  });

  it('returns false when form is clean', () => {
    setFormState({ isDirty: false, isSubmitSuccessful: false });
    const { result } = renderHook(() => useUnsavedChangesGuard());
    expect(result.current).toBe(false);
  });

  it('returns true when form is dirty', () => {
    setFormState({ isDirty: true, isSubmitSuccessful: false });
    const { result } = renderHook(() => useUnsavedChangesGuard());
    expect(result.current).toBe(true);
  });

  it('returns false when dirty but submit was successful (post-submit navigation must not be blocked)', () => {
    setFormState({ isDirty: true, isSubmitSuccessful: true });
    const { result } = renderHook(() => useUnsavedChangesGuard());
    expect(result.current).toBe(false);
  });

  it('returns false when both clean and submit successful', () => {
    setFormState({ isDirty: false, isSubmitSuccessful: true });
    const { result } = renderHook(() => useUnsavedChangesGuard());
    expect(result.current).toBe(false);
  });
});

// ── beforeunload listener lifecycle ──────────────────────────────────────────

describe('useUnsavedChangesGuard — beforeunload listener', () => {
  let addSpy: ReturnType<typeof vi.spyOn>;
  let removeSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    mockReset.mockReset();
    addSpy = vi.spyOn(window, 'addEventListener');
    removeSpy = vi.spyOn(window, 'removeEventListener');
  });

  afterEach(() => {
    addSpy.mockRestore();
    removeSpy.mockRestore();
  });

  it('registers beforeunload listener when guarded', () => {
    setFormState({ isDirty: true, isSubmitSuccessful: false });
    renderHook(() => useUnsavedChangesGuard());
    expect(addSpy).toHaveBeenCalledWith('beforeunload', expect.any(Function));
  });

  it('does not register beforeunload listener when not guarded', () => {
    setFormState({ isDirty: false, isSubmitSuccessful: false });
    renderHook(() => useUnsavedChangesGuard());
    const beforeunloadCalls = addSpy.mock.calls.filter(
      ([event]: [string]) => event === 'beforeunload'
    );
    expect(beforeunloadCalls).toHaveLength(0);
  });

  it('removes beforeunload listener when form becomes clean', () => {
    setFormState({ isDirty: true, isSubmitSuccessful: false });
    const { rerender } = renderHook(() => useUnsavedChangesGuard());

    setFormState({ isDirty: false, isSubmitSuccessful: false });
    act(() => rerender());

    expect(removeSpy).toHaveBeenCalledWith(
      'beforeunload',
      expect.any(Function)
    );
  });

  it('removes beforeunload listener when submit succeeds', () => {
    setFormState({ isDirty: true, isSubmitSuccessful: false });
    const { rerender } = renderHook(() => useUnsavedChangesGuard());

    setFormState({ isDirty: true, isSubmitSuccessful: true });
    act(() => rerender());

    expect(removeSpy).toHaveBeenCalledWith(
      'beforeunload',
      expect.any(Function)
    );
  });

  it('removes beforeunload listener on unmount', () => {
    setFormState({ isDirty: true, isSubmitSuccessful: false });
    const { unmount } = renderHook(() => useUnsavedChangesGuard());
    unmount();
    expect(removeSpy).toHaveBeenCalledWith(
      'beforeunload',
      expect.any(Function)
    );
  });
});

// ── Re-arm behaviour ──────────────────────────────────────────────────────────

describe('useUnsavedChangesGuard — re-arm after failed async mutation', () => {
  beforeEach(() => {
    mockReset.mockReset();
  });

  it('calls reset() when submitError becomes truthy after a successful sync submit', () => {
    setFormState({ isDirty: true, isSubmitSuccessful: true });

    const { rerender } = renderHook(
      (props: { submitError?: string | null }) =>
        useUnsavedChangesGuard(props.submitError),
      { initialProps: { submitError: null as string | null } }
    );

    act(() => rerender({ submitError: 'Server rejected the request' }));

    expect(mockReset).toHaveBeenCalledWith(undefined, {
      keepValues: true,
      keepDirty: true,
      keepIsSubmitted: false,
    });
  });

  it('does not call reset() when submitError is set but submit was not yet successful', () => {
    setFormState({ isDirty: true, isSubmitSuccessful: false });

    const { rerender } = renderHook(
      (props: { submitError?: string | null }) =>
        useUnsavedChangesGuard(props.submitError),
      { initialProps: { submitError: null as string | null } }
    );

    act(() => rerender({ submitError: 'Unrelated error from a prior render' }));

    expect(mockReset).not.toHaveBeenCalled();
  });

  it('re-arms guard on each consecutive submit failure even when submitError string is unchanged', () => {
    // After first re-arm reset() clears isSubmitSuccessful→false. A second
    // submit flips it back to true while submitError stays the same string.
    // The effect must fire again because isSubmitSuccessful changed.
    setFormState({ isDirty: true, isSubmitSuccessful: false });

    const { rerender } = renderHook(
      (props: { submitError?: string | null }) =>
        useUnsavedChangesGuard(props.submitError),
      { initialProps: { submitError: null as string | null } }
    );

    // First failure: isSubmitSuccessful flips to true, submitError arrives
    setFormState({ isDirty: true, isSubmitSuccessful: true });
    act(() => rerender({ submitError: 'Server error' }));
    expect(mockReset).toHaveBeenCalledTimes(1);

    // Simulate the reset() having cleared isSubmitSuccessful — must rerender so
    // the effect sees the intermediate false value before the next true flip.
    mockReset.mockReset();
    setFormState({ isDirty: true, isSubmitSuccessful: false });
    act(() => rerender({ submitError: 'Server error' }));
    expect(mockReset).not.toHaveBeenCalled();

    // Second submit: isSubmitSuccessful flips to true again, same error string
    setFormState({ isDirty: true, isSubmitSuccessful: true });
    act(() => rerender({ submitError: 'Server error' }));
    expect(mockReset).toHaveBeenCalledTimes(1);
  });
});
