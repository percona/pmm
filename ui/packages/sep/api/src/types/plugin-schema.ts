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
  /**
   * Consequence text for a field whose enabled or set state irreversibly
   * destroys user data. Presence is the mark — there is no separate boolean,
   * so `if (field.destructive)` is the check, and the string is what a
   * confirmation displays. Unmarked fields either omit the key or send it as
   * null, depending on whether the serving route excludes nulls, so test
   * truthiness rather than presence.
   */
  destructive?: string | null;
  /**
   * Name of a sibling `bool` field in the same section that this field
   * parameterises. The renderer draws the field indented beneath that parent
   * and keeps it non-interactive until the parent is on, instead of hiding it
   * — a reader can see what enabling the parent will offer.
   *
   * Presentation only, and the renderer takes it on trust: the disable state
   * comes from the named field's truthiness alone. Enforcement stays with the
   * backend, through this field's own `forbidden` gate on the parent being
   * falsy — which the renderer recognises structurally and consumes as the
   * disable condition rather than applying it as a hide. Every other gate on
   * the field keeps hiding it as usual, so declaring `parent` without that
   * companion gate leaves the value unguarded server-side; a dev build warns.
   *
   * Must name a `bool` field in the same section, declared before this one.
   * Chains and cycles are unsupported — a cycle leaves both toggles inert.
   */
  parent?: string;
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
  /**
   * Whether this section holds expert options rather than the common case.
   *
   * Advanced sections are withheld behind a single "Show advanced options"
   * control rendered after the ordinary ones, so a form with many expert
   * sections costs one row at rest instead of one per section. Revealing them
   * renders each as an ordinary top-level section — deliberately not nested
   * inside a wrapper, which reads as one disclosure level too many.
   *
   * The renderer reveals them on its own, and expands the section concerned,
   * whenever one holds a value other than its schema default or a field
   * carrying a validation error: an option someone has already set must never
   * be hidden from them. In practice the error case means a backend 422 — a
   * field inside a section that was never opened is never registered, so
   * client-side validation cannot flag it. Order is preserved, and advanced
   * sections render after the ordinary ones wherever they sit in `forms`.
   */
  advanced?: boolean;
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
  /** Optional map from a raw cell value to the text to display in its place. Absent when the app declares no labels; a value missing from the map renders as-is. */
  value_labels?: Record<string, string>;
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
  /** Optional map from a raw resolved value to the text to display in its place. Absent when the app declares no labels; a value missing from the map renders as-is. */
  value_labels?: Record<string, string>;
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
  /** What one record of this entity is called, in mid-sentence form — capitalise
   * the first character when it opens a label. */
  item_display_name: string;
  /** What several records of this entity are called, same convention. */
  item_display_name_plural: string;
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

// ── Task status vocabulary ──────────────────────────────────────────────

/**
 * One task-status value and whether it ends a run. A client polling a task to
 * completion re-reads until the row reaches a status whose `terminal` is true.
 */
export interface TaskStatusDescriptor {
  /** A `TaskHistoryStatusEnum` member, deliberately widened to `string` here
   * rather than typed as a literal union like `ColumnFormat`: the point of
   * publishing this list is that a client discovers the vocabulary at runtime
   * instead of hardcoding it. The generated client in `generated/sep.ts`
   * narrows the same field to a union of the current members, so a consumer
   * that wants runtime discovery should read this type rather than that one. */
  value: string;
  terminal: boolean;
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
  /** What one record this app's create form produces is called, in mid-sentence
   * form — capitalise the first character when it opens a label. */
  item_display_name: string;
  /** What several such records are called, same convention. */
  item_display_name_plural: string;
  /** Status vocabulary for task-style apps; omitted when `entities` is set. */
  task_statuses?: TaskStatusDescriptor[];
}
