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

import type {
  FormSection,
  OneOfGroup,
  PluginField,
  SectionField,
} from '@sep/api';

/** Type guard for the `one_of` section container. */
export function isOneOfGroup(field: SectionField): field is OneOfGroup {
  return field.type === 'one_of';
}

/** Collect every {@link OneOfGroup} container across ``sections``. */
export function collectOneOfGroups(sections: FormSection[]): OneOfGroup[] {
  return sections.flatMap((section) => section.fields.filter(isOneOfGroup));
}

/** Expand one section item to its leaf fields (pass-through or branch flatten). */
export function flattenSectionItem(field: SectionField): PluginField[] {
  if (isOneOfGroup(field)) {
    return field.branches.flatMap((branch) => branch.fields);
  }
  return [field];
}

/** Return every leaf {@link PluginField} across ``sections``, expanding one-of groups. */
export function flattenSectionFields(sections: FormSection[]): PluginField[] {
  return sections.flatMap((section) =>
    section.fields.flatMap(flattenSectionItem)
  );
}
