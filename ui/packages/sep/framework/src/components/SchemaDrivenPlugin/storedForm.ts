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
 * Reserved key under a task's ``data`` that holds the verbatim create-form body
 * the backend stamps on the JSON create path. Mirrors the backend
 * ``RESERVED_FORM_KEY`` in ``app/sep/plugins/framework/spec.py``.
 */
export const STORED_FORM_KEY = '_form';

/**
 * Read a task's stored create-form body from ``data[_form]``, centralising the
 * cast and presence guard.
 *
 * The typed client exposes ``data`` as opaque (``Record<string, never>`` in
 * codegen, ``unknown`` at the loose hook call site), so consumers route through
 * this guard rather than indexing ``data`` directly. Returns ``undefined`` for a
 * task with no stored form — a legacy task or one created through a still-live
 * legacy form — so the Edit affordance can treat the key as optional.
 */
export function getStoredForm(
  task: Record<string, unknown> | null | undefined
): Record<string, unknown> | undefined {
  const data = task?.data;
  if (!data || typeof data !== 'object') {
    return undefined;
  }
  const stored = (data as Record<string, unknown>)[STORED_FORM_KEY];
  if (!stored || typeof stored !== 'object' || Array.isArray(stored)) {
    return undefined;
  }
  return stored as Record<string, unknown>;
}
