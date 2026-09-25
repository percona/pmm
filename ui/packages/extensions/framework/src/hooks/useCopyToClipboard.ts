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

import { useCallback, useEffect, useRef, useState } from 'react';

/** How long a successful copy keeps reporting itself, in milliseconds. */
const COPIED_FEEDBACK_MS = 2000;

export interface CopyToClipboard {
  /** Copy `text`, reporting whether it landed. */
  copy: (text: string) => Promise<boolean>;
  /** True for a short window after a successful copy, for button feedback. */
  copied: boolean;
  /**
   * True when the last attempt failed — including when the clipboard is out of
   * reach, as it is on the plain-HTTP origins PMM Server is often reached on.
   * Cleared by the next attempt.
   */
  failed: boolean;
}

/**
 * Copy text to the clipboard and report the outcome for long enough to show it.
 *
 * The `copied` flag self-clears, so a caller can swap a Copy icon for a tick
 * without owning a timer. The timer is cancelled on unmount — a viewer inside
 * an accordion that `unmountOnExit`s is closed mid-window often enough for a
 * stray `setState` to matter.
 */
export function useCopyToClipboard(): CopyToClipboard {
  const [copied, setCopied] = useState(false);
  const [failed, setFailed] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const mountedRef = useRef(true);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      if (timerRef.current !== undefined) {
        clearTimeout(timerRef.current);
      }
    };
  }, []);

  const copy = useCallback(async (text: string) => {
    let ok = false;
    // `isSecureContext` is checked as well as the API's presence: some browsers
    // expose `navigator.clipboard` on an insecure origin but reject every write.
    if (globalThis.isSecureContext && navigator.clipboard) {
      try {
        await navigator.clipboard.writeText(text);
        ok = true;
      } catch {
        ok = false;
      }
    }
    // `writeText` is a promise, and this viewer lives inside accordions that
    // unmount on collapse — so the await above can outlive the component.
    // Reporting into an unmounted tree would both warn and leave a timer
    // scheduled after the cleanup that was supposed to cancel it.
    if (!mountedRef.current) {
      return ok;
    }

    // Set on every attempt, so a success after a failure clears the error state
    // rather than leaving the button red.
    setCopied(ok);
    setFailed(!ok);
    if (timerRef.current !== undefined) {
      clearTimeout(timerRef.current);
    }
    if (ok) {
      timerRef.current = setTimeout(() => setCopied(false), COPIED_FEEDBACK_MS);
    }
    return ok;
  }, []);

  return { copy, copied, failed };
}
