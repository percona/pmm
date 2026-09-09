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

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { downloadBlob } from './downloadBlob';

describe('downloadBlob', () => {
  const createObjectURL = vi.fn(() => 'blob:mock-url');
  const revokeObjectURL = vi.fn();
  let clickSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    vi.useFakeTimers();
    createObjectURL.mockClear();
    revokeObjectURL.mockClear();
    vi.stubGlobal('URL', { ...URL, createObjectURL, revokeObjectURL });
    clickSpy = vi
      .spyOn(HTMLAnchorElement.prototype, 'click')
      .mockImplementation(() => {});
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
    vi.useRealTimers();
  });

  it('clicks a hidden anchor with the blob URL and suggested filename', () => {
    const blob = new Blob(['data']);

    downloadBlob(blob, 'report.pdf');

    expect(createObjectURL).toHaveBeenCalledWith(blob);
    expect(clickSpy).toHaveBeenCalledOnce();
    const anchor = clickSpy.mock.instances[0] as HTMLAnchorElement;
    expect(anchor.href).toContain('blob:mock-url');
    expect(anchor.download).toBe('report.pdf');
    // The anchor is hidden so it never affects layout or focus order.
    expect(anchor.style.display).toBe('none');
    // Anchor is removed after the click; nothing lingers in the DOM.
    expect(document.querySelector('a')).toBeNull();
  });

  it('defers revoking the object URL until after the current tick', () => {
    downloadBlob(new Blob(['data']), 'report.pdf');

    // Not revoked synchronously — Safari/Firefox race the download otherwise.
    expect(revokeObjectURL).not.toHaveBeenCalled();
    vi.runAllTimers();
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:mock-url');
  });

  it('removes the anchor and still revokes the URL when the anchor click throws', () => {
    clickSpy.mockImplementation(() => {
      throw new Error('click failed');
    });

    expect(() => downloadBlob(new Blob(['data']), 'report.pdf')).toThrow(
      'click failed'
    );
    // The finally block cleans up even on the throw path.
    expect(document.querySelector('a')).toBeNull();
    vi.runAllTimers();
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:mock-url');
  });
});
