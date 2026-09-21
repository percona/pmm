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

import { act, render, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  flushPromises,
  mockStreamFetch,
} from '../../../../tests/eventSourceStub';
import { useTaskLogs } from '../../../hooks/useTaskLogs';
import { LogOutputPane } from '../LogOutputPane';

vi.mock('@sep/api', () => ({
  getToken: () => null,
  refreshAccessToken: vi.fn(),
  emitUnauthorized: vi.fn(),
  SEP_BASE_PATH: '/sep',
}));

function LiveLog() {
  const { textByStep, streamStatus } = useTaskLogs(7);
  return (
    <div data-status={streamStatus}>
      <LogOutputPane text={textByStep.setup?.stdout ?? ''} wrap={false} />
    </div>
  );
}

function renderedLines(container: HTMLElement): string[] {
  return Array.from(container.querySelectorAll('.log-content'), (line) =>
    // Nonempty rows include a selectable newline added by the renderer.
    (line.textContent ?? '').replace(/\n$/, '')
  );
}

describe('LogOutputPane with the real log renderer', () => {
  let mock: ReturnType<typeof mockStreamFetch>;

  beforeEach(() => {
    mock = mockStreamFetch();
    mock.install();
    vi.spyOn(HTMLElement.prototype, 'offsetParent', 'get').mockReturnValue(
      document.body
    );
    vi.stubGlobal(
      'ResizeObserver',
      class {
        constructor(private callback: ResizeObserverCallback) {}
        observe(target: Element) {
          queueMicrotask(() =>
            this.callback(
              [
                {
                  target,
                  contentRect: {
                    width: 800,
                    height: target.classList.contains('react-lazylog')
                      ? 400
                      : 19,
                  },
                } as ResizeObserverEntry,
              ],
              this
            )
          );
        }
        unobserve() {}
        disconnect() {}
      }
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it.each(['final\n', 'final\n\n', 'final', 'final\u0000'])(
    'renders the completed live log exactly like stored text ending in %j',
    async (ending) => {
      const storedText = `first\n${ending}`;
      const { container, rerender } = render(<LiveLog />);
      await act(flushPromises);

      const handle = mock.pending[0];
      act(() => {
        handle.pushMessage({
          msg: 'first\nfi',
          step: 'setup',
          type: 'stdout',
          offset: 8,
        });
      });
      await waitFor(() =>
        expect(container.querySelector('.log-content')).not.toBeNull()
      );
      act(() => {
        handle.pushMessage({
          msg: ending.slice(2),
          step: 'setup',
          type: 'stdout',
          offset: storedText.length,
        });
        handle.pushNamed('finish', { status: 'success' });
      });
      await waitFor(() =>
        expect(container.firstElementChild).toHaveAttribute(
          'data-status',
          'finished'
        )
      );

      const expectedLines = storedText.replace(/\n$/, '').split('\n');
      const liveLines = renderedLines(container);
      expect(liveLines).toEqual(expectedLines);
      expect(liveLines.at(-1)).toBe(expectedLines.at(-1));
      expect(liveLines.join('\n').match(/\u0000/g) ?? []).toHaveLength(
        (storedText.match(/\u0000/g) ?? []).length
      );

      rerender(<LogOutputPane text={storedText} wrap={false} />);
      await waitFor(() => expect(renderedLines(container)).toEqual(liveLines));
    }
  );
});
