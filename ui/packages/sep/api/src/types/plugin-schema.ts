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
 * Plugin Schema types — defines the contract for schema-driven plugins.
 *
 * These types mirror the backend PluginSchema Pydantic model.
 * The backend serves schemas at GET /api/apps/{name}/schema as JSON.
 * The SchemaFormRenderer auto-generates the UI from these definitions.
 *
 * Wire format is snake_case end-to-end. Uses a discriminated union on
 * `type` for type-safe field rendering.
 */

// ── Conditional-rule primitives (SEP-1071) ──────────────────────────────

/**
 * Predicate JSON shape — a single-key object whose key is the operator
 * name (`equals`, `truthy`, `all`, `any`, `xor`, `not`, etc.). The full
 * operator catalogue lives alongside the BE DSL; we keep this open here
 * because the FE counterpart (SEP-1077) consumes the wire format directly.
 */
export type Predicate = Record<string, unknown>;

/**
 * Binary self-cardinality gate at BaseField scope — when `when` matches,
 * the field carrying the gate is required (for `requires`) or forbidden
 * (for `forbidden`).
 */
export interface FieldGate {
  when: Predicate;
  message?: string;
}

/**
 * Cross-field cardinality constraint at FormSection or PluginSchema scope.
 * When `when` matches (omitted means always), the count of present fields
 * in `fields` must satisfy `min` / `max` (omitted bound = unbounded). The
 * backend strips `None`-valued keys from the wire payload, so optional
 * properties may be absent rather than `null`.
 */
export interface CardinalityRule {
  when?: Predicate;
  fields: string[];
  min?: number;
  max?: number;
  message?: string;
}

/**
 * Predicate-only invariant: the rule fails iff `fail_when` matches.
 * `error_fields` is an FE rendering hint pointing at the inputs to
 * highlight when the rule fires.
 */
export interface FailRule {
  fail_when: Predicate;
  error_fields: string[];
  message?: string;
}

// ── Base field ──────────────────────────────────────────────────────────

interface BaseField {
  name: string;
  label: string;
  required?: boolean;
  description?: string;
  default?: unknown;
  /** Self-cardinality gates: when matched, the field is required. */
  requires?: FieldGate[];
  /** Self-cardinality gates: when matched, the field is forbidden. */
  forbidden?: FieldGate[];
}

// ── Choice option ─────────────────────────────────────────────────────────

/**
 * One option in a choice / multi-choice field. `disabled` (and its optional
 * `disabled_reason` tooltip text) are opt-in UI hints; the backend omits both
 * from the wire while unset, so they arrive only for options that opt in.
 */
export interface ChoiceOption {
  label: string;
  value: string;
  /** Render the option non-selectable. Omitted from the wire while unset. */
  disabled?: boolean;
  /** Explanatory text shown (e.g. in a tooltip) when the option is disabled. */
  disabled_reason?: string;
}

// ── Concrete field types ────────────────────────────────────────────────

export interface StringField extends BaseField {
  type: 'string';
  min_length?: number;
  max_length?: number;
  pattern?: string;
  placeholder?: string;
}

export interface IntegerField extends BaseField {
  type: 'integer';
  ge?: number;
  le?: number;
  step?: number;
}

export interface FloatField extends BaseField {
  type: 'float';
  ge?: number;
  le?: number;
  step?: number;
}

export interface BoolField extends BaseField {
  type: 'bool';
}

export interface ChoiceField extends BaseField {
  type: 'choice';
  choices: ChoiceOption[];
}

export interface MultiChoiceField extends BaseField {
  type: 'multi_choice';
  choices: ChoiceOption[];
  min_items?: number;
  max_items?: number;
}

export interface TextAreaField extends BaseField {
  type: 'textarea';
  rows?: number;
  placeholder?: string;
}

export interface DateTimeField extends BaseField {
  type: 'datetime';
}

export interface FileField extends BaseField {
  type: 'file';
  accept?: string[];
}

export interface YamlField extends BaseField {
  type: 'yaml';
  rows?: number;
  placeholder?: string;
}

// ── Inventory-aware fields ──────────────────────────────────────────────

export interface ServiceField extends BaseField {
  type: 'service';
  service_types: string[];
  /** Offer free-text (free-solo) entry alongside the inventory options. */
  allow_custom?: boolean;
}

export interface SchemaField extends BaseField {
  type: 'schema';
  depends_on: string;
  /** Offer free-text (free-solo) entry alongside the cascaded options. */
  allow_custom?: boolean;
}

export interface TableField extends BaseField {
  type: 'table';
  depends_on: string;
  /** Offer free-text (free-solo) entry alongside the cascaded options. */
  allow_custom?: boolean;
}

export interface HostField extends BaseField {
  type: 'host';
  /**
   * Optional upstream field whose value drives the default executor
   * selection (typically a service field). Omitted when the host list is
   * not cascaded.
   */
  depends_on?: string;
  /**
   * Service field for a non-blocking co-location warning. Independent of
   * `depends_on`. Omitted when unset.
   */
  target_service?: string;
  /** Offer free-text (free-solo) entry alongside the inventory options. */
  allow_custom?: boolean;
}

// ── Read-only preview ───────────────────────────────────────────────────

export interface ScriptPreviewField extends BaseField {
  type: 'script_preview';
  /**
   * Fully-resolved URL the renderer fetches preview content from. Schema
   * synthesisers should bake plugin-specific path segments here at schema
   * build time rather than templating client-side.
   */
  endpoint_url: string;
  /**
   * Names of sibling fields whose values trigger a debounced re-fetch.
   * Empty (the default) means fetch once on mount.
   */
  depends_on: string[];
  /** Optional default highlighter language hint. */
  language?: string;
}

// ── Dynamic, API-backed option source ────────────────────────────────────

export interface RemoteChoiceField extends BaseField {
  type: 'remote_choice';
  /**
   * Fully-resolved URL the renderer fetches `Choice`-compatible options from,
   * relative to the `apiClient` base (`/api`). Schema synthesisers bake any
   * plugin-specific path segments here at schema build time.
   */
  endpoint_url: string;
  /**
   * Optional sibling field name whose value drives (and parameterises) the
   * option fetch. Omitted from the wire while unset.
   */
  depends_on?: string;
  /** Offer free-text (free-solo) entry alongside the fetched options. */
  allow_custom?: boolean;
}

// ── Discriminated union ─────────────────────────────────────────────────

export type PluginField =
  | StringField
  | IntegerField
  | FloatField
  | BoolField
  | ChoiceField
  | MultiChoiceField
  | TextAreaField
  | DateTimeField
  | FileField
  | YamlField
  | ServiceField
  | SchemaField
  | TableField
  | HostField
  | RemoteChoiceField
  | ScriptPreviewField;

// ── One-of group ─────────────────────────────────────────────────────────

/** One mutually-exclusive branch inside a {@link OneOfGroup}. */
export interface OneOfBranch {
  value: string;
  label: string;
  fields: PluginField[];
}

/**
 * Labelled either/or field group rendered as a segmented control.
 * Branch leaves use dotted paths when nested on the write model.
 */
export interface OneOfGroup {
  type: 'one_of';
  /** Stable group id for React keys; not a separate form value. */
  name: string;
  label: string;
  description?: string;
  /** Dotted path to the mode field (e.g. `source.mode`). */
  discriminator: string;
  default?: string;
  branches: OneOfBranch[];
}

/** A section item: a leaf field or a one-of group container. */
export type SectionField = PluginField | OneOfGroup;

// ── Form structure ──────────────────────────────────────────────────────

export interface FormSection {
  title: string;
  description?: string;
  fields: SectionField[];
  /** Whether the section is wrapped in an expandable/collapsible shell. */
  collapsible?: boolean;
  /** Initial expansion state when collapsible is enabled. */
  collapsed_by_default?: boolean;
  /** Whether to render the section after the submit button. */
  render_after_submit?: boolean;
  /** Cross-field cardinality constraints scoped to this section. */
  cardinality_rules?: CardinalityRule[];
  /** Predicate-only invariants scoped to this section. */
  fail_when?: FailRule[];
  /**
   * Section-level visibility gates. When any gate fires the entire section
   * is hidden and every child field is unregistered from react-hook-form
   * so stale values do not ship in the submission payload.
   * Gates may reference any field in the plugin schema.
   */
  forbidden?: FieldGate[];
}

// ── List view ───────────────────────────────────────────────────────────

export interface ListColumn {
  key: string;
  label: string;
  sortable?: boolean;
  format?:
    | 'text'
    | 'chip'
    | 'status'
    | 'date'
    | 'relative'
    | 'code'
    | 'actions'
    | 'schedule';
}

export interface ListView {
  columns: ListColumn[];
  /** Column key to sort by. Prefix with '-' for descending (e.g. '-last_run'). */
  default_sort?: string;
  /** Extra record-level keys to hide from the detail Overview — both the list_view.columns rows and the extras loop, across single-task and multi-entity detail views — merged with the framework baseline. */
  overview_hidden_fields?: string[];
}

// ── Capabilities ────────────────────────────────────────────────────────

export interface PluginCapabilities {
  chaining?: boolean;
  alert_on_fail?: boolean;
  scheduling?: boolean;
  stats?: boolean;
  pii_anonymization?: boolean;
}

// ── Detail view (task-style plugins) ────────────────────────────────────

/** One labelled field rendered inside a DetailSection. */
export interface DetailField {
  /** Dotted path into the task record (e.g. ``"data.meta.command"``). */
  path: string;
  label: string;
  /** Optional syntax-highlighter hint; mirrors the backend ``DetailHighlightLanguage`` enum. */
  highlight?: 'sql' | 'json' | 'bash' | 'yaml';
}

/** One titled section rendered on the task detail page. */
export interface DetailSection {
  title: string;
  fields: DetailField[];
}

/** Declarative layout for the task detail page's section cards. */
export interface DetailView {
  sections: DetailSection[];
}

// ── Multi-entity plugins (inventory) ────────────────────────────────────

/** One CRUD resource when a plugin exposes several (nodes, services, …). */
export interface PluginEntitySchema {
  name: string;
  display_name: string;
  description?: string;
  forms: FormSection[];
  list_view: ListView;
  /** Optional detail-view syntax hints keyed by field name; mirrors the backend
   * ``DetailHighlightLanguage`` enum. */
  detail_highlights?: Partial<Record<string, 'sql' | 'json' | 'bash' | 'yaml'>>;
}

// ── Related apps (sibling tabs) ─────────────────────────────────────────

/**
 * A separately registered app the parent plugin surfaces as a sibling tab
 * (for example `mysql_backups/restore` nested under MySQL Backups).
 */
export interface RelatedApp {
  /** Scoped registry key (for example `mysql_backups/restore`). */
  app_key: string;
  /** Tab label shown in the React shell (for example `Restore`). */
  label: string;
  /** Sub-path segment under the parent's `route_base` (for example `restores`). */
  route_segment: string;
}

// ── Top-level schema ────────────────────────────────────────────────────

export interface PluginSchema {
  name: string;
  display_name: string;
  description?: string;
  task_type?: string;
  /** Task-style single entity: forms + list_view (omit or leave entities unset). */
  forms?: FormSection[];
  capabilities?: PluginCapabilities;
  list_view?: ListView;
  /** Declarative layout for the task detail page (task-style plugins). */
  detail_view?: DetailView;
  /** When set, the shell renders one list/create/detail flow per entity. */
  entities?: PluginEntitySchema[];
  /** Schema-wide cross-field cardinality constraints (task-style plugins). */
  cardinality_rules?: CardinalityRule[];
  /** Schema-wide predicate-only invariants (task-style plugins). */
  fail_when?: FailRule[];
  /** Separately registered apps rendered as sibling tabs in the React shell. */
  related_apps?: RelatedApp[];
}
