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
 * Local type aliases for the plugin schema shape.
 *
 * Re-exports from `@sep/api` so field components and tests have a single
 * import surface. If the backend schema shape ever diverges from the api
 * package's contract, adapt here without touching every field file.
 */

import type { ReactNode } from 'react';
import type { PluginField as PluginFieldType } from '@sep/api';

/** Argument bag handed to a {@link RenderFieldOverride}. */
export interface RenderFieldArgs {
  /**
   * The gate-resolved field: visibility and the `required` flag have already
   * been decided by the conditional engine, so an override sees the same field
   * the framework would render.
   */
  field: PluginFieldType;
  /** Render the framework default widget (the existing `FieldRenderer`) for this field. */
  renderDefault: () => ReactNode;
}

/**
 * Per-field widget override for {@link SchemaFormRenderer}.
 *
 * Return custom UI for the field, call `renderDefault()`, or return a nullish
 * value (`undefined` / `null`) to fall back to the framework widget — so an
 * override keyed by field name can handle some fields and defer the rest. The
 * override changes the widget, not the rules: a field
 * hidden by its gate never reaches the override, and the resolved `required`
 * flag is already applied to `args.field`.
 *
 * Write-through contract: an override MUST persist its value through
 * react-hook-form (`useFormContext` / `Controller` / `register`). The
 * `fail_when` and cardinality rule engines read react-hook-form `watch` state,
 * as does form submission; an override that keeps its own local state instead
 * silently breaks both. Memoize expensive overrides — the slot is wrapped in
 * `memo`, so an unstable override identity re-renders it on every parent render.
 */
export type RenderFieldOverride = (args: RenderFieldArgs) => ReactNode;

export type {
  PluginSchema,
  PluginField,
  SectionField,
  OneOfBranch,
  OneOfGroup,
  FormSection,
  StringField,
  IntegerField,
  FloatField,
  BoolField,
  ChoiceField,
  ChoiceOption,
  MultiChoiceField,
  TextAreaField,
  DateTimeField,
  FileField,
  YamlField,
  ServiceField,
  SchemaField,
  TableField,
  HostField,
  RemoteChoiceField,
  ScriptPreviewField,
  Predicate,
  FieldGate,
  CardinalityRule,
  FailRule,
} from '@sep/api';
