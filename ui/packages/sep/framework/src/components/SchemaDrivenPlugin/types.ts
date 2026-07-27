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

import type { ReactNode } from 'react';
import type { PluginCapabilities, FieldValidationError } from '@sep/api';
import type {
  FormSection,
  RenderFieldOverride,
} from '../SchemaFormRenderer/types';

/**
 * Everything a whole-form slot needs to render and submit a plugin form. The
 * framework still owns the route, page chrome (back navigation + title),
 * mutation wiring, and success / error snackbars; only the form body is
 * replaced. A slot composes {@link SchemaFormRenderer} internally with these
 * props, or builds its own form.
 *
 * Submit contract: the slot MUST call `onSubmit` (the framework submit handler)
 * rather than firing its own mutation, or the success / error snackbars and
 * post-submit navigation desync.
 */
export interface PluginFormSlotProps {
  /** Form sections to render (from the task / entity schema). */
  sections: FormSection[];
  /** Framework submit handler — wires the create / update mutation, snackbars, and navigation. */
  onSubmit: (data: Record<string, unknown>) => void;
  /** Whether the underlying mutation is in flight. */
  loading: boolean;
  /** Initial form values (populated for edit; undefined for create). */
  defaultValues?: Record<string, unknown>;
  /** Plugin capabilities (e.g. `alert_on_fail`) for the composed renderer. */
  capabilities?: PluginCapabilities;
  /** Per-field override threaded through, so a composed renderer can honour it. */
  renderField?: RenderFieldOverride;
  /** Form-level submit error banner (populated on a 422); pass to the composed renderer. */
  submitError?: string | null;
  /** Per-field validation errors (from a 422); pass to the composed renderer for inline display. */
  fieldErrors?: FieldValidationError[];
}

/**
 * Whole-form slot override for the create / edit pages. Return custom form UI;
 * the framework keeps the surrounding chrome, mutation, and snackbars.
 */
export type RenderFormSlot = (props: PluginFormSlotProps) => ReactNode;
