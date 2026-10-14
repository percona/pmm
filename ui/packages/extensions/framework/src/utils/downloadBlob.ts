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

/**
 * Trigger a browser download for an in-memory Blob.
 *
 * Writes the blob to an object URL, clicks a hidden anchor to start the
 * download, then defers revoking the URL. Callers handle the fetch; this owns
 * everything after the Blob is in hand.
 *
 * @param data Blob to download.
 * @param filename Suggested file name for the saved download.
 */
export function downloadBlob(data: Blob, filename: string): void {
  const url = URL.createObjectURL(data);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  // Keep the anchor out of the layout/focus order; it only exists to be clicked.
  anchor.style.display = 'none';
  document.body.appendChild(anchor);
  try {
    anchor.click();
  } finally {
    anchor.remove();
    // Defer revocation: Safari and Firefox can race the anchor-triggered
    // download if the blob URL is revoked in the same tick as .click().
    setTimeout(() => URL.revokeObjectURL(url), 0);
  }
}
