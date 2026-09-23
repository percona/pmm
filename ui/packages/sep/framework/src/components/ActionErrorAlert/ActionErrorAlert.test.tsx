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

import { act, render, renderHook, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { ApiError } from '@sep/api';
import { ActionErrorAlert } from './ActionErrorAlert';
import {
  actionErrorMessage,
  DEFAULT_ACTION_ERROR_FALLBACK,
} from './actionErrorMessage';
import { useActionError } from './useActionError';

function http(status: number, message: string, data?: unknown): ApiError {
  return new ApiError({ kind: 'http', status, message, data });
}

describe('actionErrorMessage', () => {
  it("uses the refusal's own reason for a 403", () => {
    expect(
      actionErrorMessage(
        http(403, "You don't have permission to perform this action")
      )
    ).toBe("You don't have permission to perform this action");
  });

  it('reads the per-field detail array of a 422 rather than the HTTP 422 fallback', () => {
    const error = http(422, 'HTTP 422', {
      detail: [
        { loc: ['body', 'limit'], msg: 'ensure this value is greater than 0' },
        { loc: ['body', 'task_name'], msg: 'field required' },
      ],
    });

    expect(actionErrorMessage(error)).toBe(
      'limit: ensure this value is greater than 0; task_name: field required'
    );
  });

  it('ignores an array detail on any status other than 422', () => {
    // A batch endpoint listing per-item failures still has a usable message of
    // its own; only a 422 means the array is the per-field validation shape.
    const error = http(400, 'Batch approval failed for 2 snippets', {
      detail: [{ loc: ['check.sh'], msg: 'missing on disk' }],
    });

    expect(actionErrorMessage(error)).toBe(
      'Batch approval failed for 2 snippets'
    );
  });

  it('reports transport failures with their own message', () => {
    expect(
      actionErrorMessage(
        new ApiError({ kind: 'network', message: 'Network error' })
      )
    ).toBe('Network error');
    expect(
      actionErrorMessage(
        new ApiError({ kind: 'timeout', message: 'Request timed out' })
      )
    ).toBe('Request timed out');
  });

  it('falls back only when nothing carries a message', () => {
    expect(actionErrorMessage(new Error(''))).toBe(
      DEFAULT_ACTION_ERROR_FALLBACK
    );
    expect(actionErrorMessage({}, 'Delete failed')).toBe('Delete failed');
    expect(actionErrorMessage(null)).toBe(DEFAULT_ACTION_ERROR_FALLBACK);
  });
});

describe('ActionErrorAlert', () => {
  it('renders nothing without an error', () => {
    const { container } = render(<ActionErrorAlert error={null} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("renders the server's reason", () => {
    render(<ActionErrorAlert error={http(409, 'Sync already running')} />);
    expect(screen.getByTestId('action-error-alert')).toHaveTextContent(
      'Sync already running'
    );
  });

  it('offers dismissal when a handler is given', async () => {
    const onClose = vi.fn();
    render(<ActionErrorAlert error={new Error('boom')} onClose={onClose} />);

    await userEvent.click(screen.getByRole('button', { name: /close/i }));

    expect(onClose).toHaveBeenCalled();
  });
});

describe('useActionError', () => {
  it('holds a reported failure until it is cleared', () => {
    const { result } = renderHook(() => useActionError());
    expect(result.current.message).toBeNull();

    act(() => result.current.reportError(http(403, 'Refused')));
    expect(result.current.message).toBe('Refused');

    act(() => result.current.clearError());
    expect(result.current.message).toBeNull();
  });

  it('never reports a failure as no failure', () => {
    const { result } = renderHook(() => useActionError('Delete failed'));

    act(() => result.current.reportError(null));

    expect(result.current.message).toBe('Delete failed');
  });
});
